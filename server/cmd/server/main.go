package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
	"xiezuojun-server/internal/auth"
	"xiezuojun-server/internal/gateway"
	"xiezuojun-server/internal/middleware"
	"xiezuojun-server/internal/memory"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/project"
	"xiezuojun-server/internal/user"
)

func main() {
	_ = godotenv.Load()

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Fatal("DB_DSN not set")
	}

	db, err := gorm.Open(mysql.Open(dbDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.Project{},
		&model.Document{},
		&model.DocumentVersion{},
		&model.WordCountLog{},
	); err != nil {
		log.Fatal("auto migrate failed:", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production"
	}
	jwtManager := auth.NewJWTManager(jwtSecret)

	dataRoot := os.Getenv("DATA_ROOT")
	if dataRoot == "" {
		dataRoot = "./data"
	}

	// handlers
	userHandler := user.NewHandler(db, jwtManager)
	projectHandler := project.NewHandler(db, dataRoot)

	// AI gateway — 传入 MemoryStorage 作为 MemoryReader
	memStorage := memory.NewMemoryStorage(dataRoot)
	memHandler := memory.NewHandler(memStorage, db)

	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		apiKey = "sk-test"
	}
	aiClient := ai.NewClient(apiKey, os.Getenv("AI_BASE_URL"))
	gw := gateway.NewGateway(aiClient, memStorage)

	// 设置文档内容加载器（供 AI generate 时读取文档前文）
	gw.SetDocContentLoader(func(docID uint64) (string, error) {
		return projectHandler.LoadDocContent(uint(docID))
	})

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 公开路由
	api := r.Group("/api")
	api.POST("/auth/register", userHandler.Register)
	api.POST("/auth/login", userHandler.Login)
	api.POST("/auth/refresh", userHandler.Refresh)

	// 需要认证
	authd := api.Group("")
	authd.Use(middleware.JWTAuth(jwtManager))

	authd.GET("/user/me", userHandler.Me)
	authd.PATCH("/user/me", userHandler.UpdateProfile)
	authd.GET("/user/usage", userHandler.UsageStats)

	authd.GET("/projects", projectHandler.ListProjects)
	authd.POST("/projects", projectHandler.CreateProject)
	authd.PATCH("/projects/:id", projectHandler.UpdateProject)
	authd.DELETE("/projects/:id", projectHandler.DeleteProject)

	authd.GET("/documents", projectHandler.ListDocuments)
	authd.POST("/documents", projectHandler.CreateDocument)
	authd.GET("/documents/:id", projectHandler.GetDocument)

	// 文档更新：包装原 handler，成功后异步提取记忆
	authd.PATCH("/documents/:id", func(c *gin.Context) {
		// 缓存请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "read body failed"})
			return
		}

		// 恢复 body 供原 handler 读取
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		// 调用原 handler
		projectHandler.UpdateDocument(c)

		// 如果更新成功，异步触发章节提取
		if c.Writer.Status() == 200 {
			var req struct {
				Content string `json:"content"`
			}
			if json.Unmarshal(body, &req) != nil || req.Content == "" {
				return
			}

			idStr := c.Param("id")
			docID, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				return
			}

			userIDStr := ""
			if val, exists := c.Get("userID"); exists {
				userIDStr = fmt.Sprintf("%d", val.(uint))
			}

			var doc model.Document
			if err := db.First(&doc, docID).Error; err != nil {
				return
			}
			pid := fmt.Sprintf("%d", doc.ProjectID)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[main] extract panic: %v", r)
					}
				}()
				memory.ExtractAndSaveSummary(
					c.Request.Context(),
					aiClient,
					memStorage,
					userIDStr,
					pid,
					idStr,
					req.Content,
				)
			}()
		}
	})

	authd.DELETE("/documents/:id", projectHandler.DeleteDocument)

	// ===== 记忆系统 API =====
	authd.GET("/memory/files", memHandler.ListFiles)
	authd.GET("/memory/file", memHandler.GetFile)
	authd.PUT("/memory/file", memHandler.WriteFile)
	authd.DELETE("/memory/file", memHandler.DeleteFile)

	authd.GET("/memory/characters", memHandler.ListCharacters)
	authd.GET("/memory/characters/:name", memHandler.GetCharacter)
	authd.PUT("/memory/characters/:name", memHandler.WriteCharacter)
	authd.DELETE("/memory/characters/:name", memHandler.DeleteCharacter)

	authd.GET("/memory/summaries", memHandler.ListSummaries)
	authd.GET("/memory/summaries/:chapter_id", memHandler.GetSummary)
	authd.PUT("/memory/summaries/:chapter_id", memHandler.WriteSummary)

	// AI 路由
	aiGroup := authd.Group("")
	aiGroup.Use(gw.RateLimitMiddleware())
	aiGroup.POST("/ai/generate", gw.HandleGenerate)
	aiGroup.GET("/ai/models", gw.HandleGetModels)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 写作菌后端启动，端口:", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("server failed:", err)
	}
}
