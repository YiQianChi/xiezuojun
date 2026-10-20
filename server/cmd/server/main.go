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
	"xiezuojun-server/internal/buddy"
	"xiezuojun-server/internal/gateway"
	"xiezuojun-server/internal/memory"
	"xiezuojun-server/internal/middleware"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/project"
	"xiezuojun-server/internal/store"
	"xiezuojun-server/internal/tts"
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
		&model.Project{},
		&model.StoryBranch{},
		&model.Document{},
		&model.DocumentVersion{},
		&model.WordCountLog{},
		&model.UsageRecord{},
		&model.StoryThread{},
	); err != nil {
		log.Fatal("auto migrate failed:", err)
	}


	// ponytail: one-shot migration — wrap orphaned setting docs into a "设定" folder
	migrateSettingFolder(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" || jwtSecret == "change-me-in-production" || len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set to a strong value (>=32 chars); refusing to start")
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
	pathConfig := &store.PathConfig{DocRoot: docRoot}
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

	memHandler := memory.NewHandler(db, docRoot)
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

	// 设置伏笔加载器（AI 上下文注入未回收的伏笔/钩子）
	gw.SetThreadLoader(func(projectID uint) (string, error) {
		threads, err := projectHandler.LoadOpenThreads(projectID)
		if err != nil {
			return "", err
		}
		return project.FormatThreadsForAI(threads), nil
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
	authd.GET("/projects/:id/export/epub", projectHandler.ExportEPUB)
	authd.GET("/projects/:id/export/txt", projectHandler.ExportTXT)
	authd.GET("/projects/:id/export/docx", projectHandler.ExportDOCX)

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

		// 如果更新成功，异步触发记忆提取（仅人物设定文档提取角色）
		if c.Writer.Status() == 200 {
			var req struct {
				Content string `json:"content"`
			}
			if json.Unmarshal(body, &req) != nil || req.Content == "" {
				return
			}

			idStr := c.Param("id")
			docID64, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				return
			}
			docID := uint(docID64)
			userIDStr := ""
			if val, exists := c.Get("userID"); exists {
				userIDStr, _ = val.(string)
			}

			var doc model.Document
			if err := db.First(&doc, docID).Error; err != nil {
				return
			}

			if doc.Type == "characters" {
				pid := fmt.Sprintf("%d", doc.ProjectID)
				uid := userIDStr
				pidStr := pid
				content := req.Content
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[main] extract panic: %v", r)
						}
					}()
					memory.ExtractCharacters(
						context.Background(),
						aiClient,
						memStorage,
						uid,
						pidStr,
						content,
					)
				}()
			} else if doc.Type == "chapter" {
				// ponytail: 复用同一段异步提取骨架，差异只在调用的函数
				pid := fmt.Sprintf("%d", doc.ProjectID)
				docIDStr := strconv.FormatUint(uint64(doc.ID), 10)
				title, content := doc.Title, req.Content
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[main] summary panic: %v", r)
						}
					}()
					memory.ExtractChapterSummary(
						context.Background(),
						aiClient,
						memStorage,
						userIDStr,
						pid,
						docIDStr,
						title,
						content,
					)
				}()
			}
		}
	})

	authd.DELETE("/documents/:id", projectHandler.DeleteDocument)

	// 回收站路由
	authd.GET("/trash", projectHandler.TrashListHandler)
	authd.POST("/trash/restore", projectHandler.RestoreDocument)
	authd.DELETE("/trash", projectHandler.EmptyTrash)
	authd.POST("/trash/destroy", projectHandler.DestroyDocument)

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
	authd.GET("/memory/relationships", memHandler.GetRelationships)
	authd.PUT("/memory/relationships", memHandler.PutRelationships)

	authd.GET("/memory/:project_id", memHandler.GetProjectMemory)
	authd.PUT("/memory/:project_id/:file", memHandler.UpdateMemoryFile)
	authd.POST("/memory/:project_id/extract", memHandler.ExtractMemory)
	authd.POST("/memory/:project_id/distill", memHandler.DistillMemory)

	// AI 路由
	aiGroup := authd.Group("")
	aiGroup.Use(gw.RateLimitMiddleware())
	aiGroup.POST("/ai/generate", gw.HandleGenerate)
	aiGroup.POST("/ai/chat", gw.HandleChat)
	aiGroup.GET("/ai/models", gw.HandleGetModels)

	// TTS 朗读路由
	authd.GET("/tts/voices", tts.HandleVoices)
	authd.POST("/tts", tts.HandleTTS)

	// 搭子（结伴写作）路由
	buddy.RegisterRoutes(authd)

	// 伏笔/钩子追踪
	authd.GET("/projects/:id/stats", projectHandler.WritingStats)
	authd.GET("/projects/:id/threads", projectHandler.ListThreads)
	authd.POST("/projects/:id/threads", projectHandler.CreateThread)
	authd.PATCH("/threads/:id", projectHandler.UpdateThread)
	authd.DELETE("/threads/:id", projectHandler.DeleteThread)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 写作菌后端启动，端口:", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("server failed:", err)
	}
}

// migrateSettingFolder: 对每个项目，如果设定类文档（outline/characters/worldbuilding/storyline/chapter_outline）
// 是根级（parent_id IS NULL），则创建一个"设定"文件夹把它们包进去。已迁移过的项目跳过。
func migrateSettingFolder(db *gorm.DB) {
	settingTypes := []string{"outline", "characters", "worldbuilding", "storyline", "chapter_outline"}

	// 找所有有根级设定文档的项目
	var projectIDs []uint
	db.Table("documents").
		Where("type IN ? AND parent_id IS NULL AND status = 1", settingTypes).
		Distinct("project_id").
		Pluck("project_id", &projectIDs)

	if len(projectIDs) == 0 {
		return
	}

	for _, pid := range projectIDs {
		// 创建"设定"文件夹
		folder := model.Document{
			ProjectID: pid,
			Type:      "folder",
			Title:     "设定",
			SortOrder: 0,
			Status:    1,
		}
		if err := db.Create(&folder).Error; err != nil {
			log.Printf("迁移: 创建设定文件夹失败 project=%d: %v", pid, err)
			continue
		}

		// 把根级设定文档挂到文件夹下
		result := db.Model(&model.Document{}).
			Where("project_id = ? AND type IN ? AND parent_id IS NULL", pid, settingTypes).
			Updates(map[string]interface{}{
				"parent_id":  folder.ID,
				"sort_order": gorm.Expr("FIND_IN_SET(type, 'outline,characters,worldbuilding,storyline,chapter_outline') - 1"),
			})
		if result.Error != nil {
			log.Printf("迁移: 更新设定文档失败 project=%d: %v", pid, result.Error)
			continue
		}

		// "文章"根文件夹 sort_order 设为 1（在"设定"后面）
		db.Model(&model.Document{}).
			Where("project_id = ? AND type = 'volume' AND parent_id IS NULL", pid).
			Update("sort_order", 1)

		log.Printf("迁移: project=%d 设定文件夹已创建(id=%d)，%d 个文档已归位", pid, folder.ID, result.RowsAffected)
	}
}
