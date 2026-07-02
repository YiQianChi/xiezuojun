package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
	"xiezuojun-server/internal/auth"
	"xiezuojun-server/internal/gateway"
	"xiezuojun-server/internal/memory"
	"xiezuojun-server/internal/middleware"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/project"
	"xiezuojun-server/internal/store"
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
		&model.AuthIdentity{},
		&model.VerificationCode{},
		&model.Project{},
		&model.StoryBranch{},
		&model.Document{},
		&model.DocumentVersion{},
		&model.WordCountLog{},
		&model.UsageRecord{},
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

	// 初始化存储层
	docRoot := os.Getenv("DOC_ROOT")
	if docRoot == "" {
		docRoot = filepath.Join(dataRoot, "documents")
	}
	memRoot := os.Getenv("MEMORY_ROOT")
	if memRoot == "" {
		memRoot = filepath.Join(dataRoot, "memory")
	}
	pathConfig := &store.PathConfig{DocRoot: docRoot, MemRoot: memRoot}
	userLocks := &store.UserLocks{}

	// handlers
	userHandler := user.NewHandler(db, jwtManager)
	projectHandler := project.NewHandler(db, dataRoot, pathConfig, userLocks)
	projectHandler.MigrateFolderFiles()
	go projectHandler.RebuildMissingSearchText()

	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		apiKey = "sk-test"
	}
	aiClient := ai.NewClient(apiKey, os.Getenv("AI_BASE_URL"))

	memHandler := memory.NewHandler(db, memRoot)
	memHandler.SetAIClient(aiClient)
	memStorage := memHandler.Storage()
	gw := gateway.NewGateway(aiClient, memStorage)
	gw.SetDB(db)

	// 设置文档内容加载器（供 AI generate 时读取文档前文）
	gw.SetDocContentLoader(func(docID uint64) (string, error) {
		return projectHandler.LoadDocContent(uint(docID))
	})

	// 设置项目设定文档加载器（AI 上下文注入大纲/人物/世界观）
	gw.SetSettingLoader(func(projectID uint) ([]gateway.SettingDoc, error) {
		docs, err := projectHandler.LoadSettingDocs(projectID)
		if err != nil {
			return nil, err
		}
		result := make([]gateway.SettingDoc, 0, len(docs))
		for _, d := range docs {
			result = append(result, gateway.SettingDoc{
				ID:      d.ID,
				Title:   d.Title,
				Type:    d.Type,
				Content: d.Content,
			})
		}
		return result, nil
	})

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 公开路由
	api := r.Group("/api")
	api.POST("/auth/email-code", userHandler.SendEmailCode)
	api.POST("/auth/register", userHandler.Register)
	api.POST("/auth/login", userHandler.Login)
	api.POST("/auth/refresh", userHandler.Refresh)
	api.POST("/auth/reset-password", userHandler.ResetPassword)

	// 需要认证
	authd := api.Group("")
	authd.Use(middleware.JWTAuth(jwtManager))

	authd.GET("/user/me", userHandler.Me)
	authd.PATCH("/user/me", userHandler.UpdateProfile)
	authd.GET("/user/usage", userHandler.UsageStats)

	authd.GET("/projects", projectHandler.ListProjects)
	authd.POST("/projects", projectHandler.CreateProject)
	authd.GET("/projects/search", projectHandler.SearchProjects)
	authd.GET("/projects/:id/search", projectHandler.SearchDocuments)
	authd.GET("/projects/:id", projectHandler.GetProject)
	authd.PATCH("/projects/:id", projectHandler.UpdateProject)
	authd.DELETE("/projects/:id", projectHandler.DeleteProject)
	authd.GET("/projects/:id/branches", projectHandler.ListStoryBranches)
	authd.POST("/projects/:id/branches", projectHandler.CreateStoryBranch)
	authd.PATCH("/story-branches/:branch_id", projectHandler.UpdateStoryBranch)
	authd.GET("/projects/:id/export", projectHandler.ExportProject)

	authd.GET("/documents", projectHandler.ListDocuments)
	authd.POST("/documents", projectHandler.CreateDocument)
	authd.GET("/documents/:id", projectHandler.GetDocument)
	authd.GET("/documents/:id/versions", projectHandler.ListDocumentVersions)
	authd.GET("/document-versions/:version_id", projectHandler.GetDocumentVersion)
	authd.POST("/document-versions/:version_id/restore", projectHandler.RestoreDocumentVersion)

	authd.PATCH("/documents/:id/move", projectHandler.MoveDocument)

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
				userIDStr, _ = val.(string)
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
					context.Background(),
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

	// 回收站路由
	authd.GET("/trash", projectHandler.TrashListHandler)
	authd.POST("/trash/restore", projectHandler.RestoreDocument)
	authd.DELETE("/trash", projectHandler.EmptyTrash)

	// 回收站定时清理（每小时一次）
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[trash-cleanup] panic: %v", r)
					}
				}()
				// 遍历所有活跃用户做清理
				var userIDs []string
				if err := db.Model(&model.User{}).Pluck("id", &userIDs).Error; err != nil {
					return
				}
				for _, uid := range userIDs {
					n, err := store.TrashCleanup(pathConfig, uid)
					if err != nil {
						log.Printf("[trash-cleanup] uid=%s err=%v", uid, err)
					}
					if n > 0 {
						log.Printf("[trash-cleanup] uid=%s cleaned=%d", uid, n)
					}
				}
			}()
		}
	}()

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

	// 三层记忆
	authd.GET("/memory/:project_id", memHandler.GetProjectMemory)
	authd.PUT("/memory/:project_id/:file", memHandler.UpdateMemoryFile)
	authd.POST("/memory/:project_id/extract", memHandler.ExtractMemory)
	authd.POST("/memory/:project_id/distill", memHandler.DistillMemory)

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
