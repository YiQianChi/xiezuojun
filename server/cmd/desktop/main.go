// 写作菌桌面版入口：单文件 exe，数据落本机 SQLite。
// 与 cmd/server 的差异：
//   - SQLite（modernc pure-Go 驱动，免 CGO，可交叉编译 Windows）
//   - DESKTOP=1：Redis 依赖换成内存 miniredis；登录接口直接发 token
//   - 首次启动种一个本地账号；随机空闲端口 + 自动开浏览器
//   - 自带静态文件伺服（exe 旁的 web/ 目录）
//
// 构建产物默认不进生产部署链路，只随 GitHub Actions 出 Windows 包。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	_sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
	"xiezuojun-server/internal/auth"
	"xiezuojun-server/internal/buddy"
	"xiezuojun-server/internal/sprint"
	"xiezuojun-server/internal/sticky"
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

	// 桌面模式标记：buddy/vcode 的 Redis init 会据此切 miniredis
	_ = os.Setenv("DESKTOP", "1")
	gin.SetMode(gin.ReleaseMode)

	// 数据目录：exe 旁边的 data/（便携版语义——拷走整个文件夹=带走全部数据）
	exePath, _ := os.Executable()
	dataRoot := filepath.Join(filepath.Dir(exePath), "data")
	_ = os.MkdirAll(dataRoot, 0o755)

	db, err := gorm.Open(_sqlite.Open(filepath.Join(dataRoot, "xzj.db")), &gorm.Config{})
	if err != nil {
		log.Fatal("打开本地数据库失败:", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.RefreshToken{}, &model.AuthIdentity{},
		&model.Project{}, &model.StoryBranch{}, &model.Document{}, &model.DocumentVersion{},
		&model.WordCountLog{}, &model.UsageRecord{}, &model.StoryThread{},
		&model.WritingSession{},
		&sticky.Note{},
	); err != nil {
		log.Fatal("本地建表失败:", err)
	}

	// 本地免登录：固定 JWT 密钥（仅本机 loopback 使用，无安全边界问题）
	// ponytail-desktop: secret 存 data/ 里，删目录后 token 失效自动重发
	secretPath := filepath.Join(dataRoot, "jwt.secret")
	jwtSecret, err := os.ReadFile(secretPath)
	if err != nil || len(jwtSecret) < 32 {
		jwtSecret = []byte(uuid.NewString() + uuid.NewString())
		_ = os.WriteFile(secretPath, jwtSecret, 0o600)
	}
	jwtManager := auth.NewJWTManager(string(jwtSecret))

	// 首次启动：种本地作者账号（前端有登录态语义，给它一个真实存在的用户）
	localUID := seedLocalUser(db)

	docRoot := filepath.Join(dataRoot, "documents")
	pathConfig := &store.PathConfig{DocRoot: docRoot}
	userLocks := &store.UserLocks{}

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
	gw.SetDocContentLoader(func(docID uint64) (string, error) {
		return projectHandler.LoadDocContent(uint(docID))
	})
	gw.SetSettingLoader(func(projectID uint) ([]gateway.SettingDoc, error) {
		docs, err := projectHandler.LoadSettingDocs(projectID)
		if err != nil {
			return nil, err
		}
		result := make([]gateway.SettingDoc, 0, len(docs))
		for _, d := range docs {
			result = append(result, gateway.SettingDoc{ID: d.ID, Title: d.Title, Type: d.Type, Content: d.Content})
		}
		return result, nil
	})
	gw.SetThreadLoader(func(projectID uint) (string, error) {
		threads, err := projectHandler.LoadOpenThreads(projectID)
		if err != nil {
			return "", err
		}
		return project.FormatThreadsForAI(threads), nil
	})

	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "mode": "desktop"})
	})

	api := r.Group("/api")
	api.POST("/auth/login", func(c *gin.Context) {
		// ponytail-desktop: 桌面版登录接口=直接发 token，任何输入都通过
		token, _, _ := jwtManager.GeneratePair(localUID, "local@desktop.xiezuojun", "svip")
		c.JSON(200, gin.H{"data": gin.H{"access_token": token, "refresh_token": "desktop"}})
	})
	api.POST("/auth/refresh", func(c *gin.Context) {
		token, _, _ := jwtManager.GeneratePair(localUID, "local@desktop.xiezuojun", "svip")
		c.JSON(200, gin.H{"data": gin.H{"access_token": token, "refresh_token": "desktop"}})
	})
	api.POST("/auth/email-code", func(c *gin.Context) {
		c.JSON(400, gin.H{"error": "桌面版无需邮箱验证"})
	})
	api.POST("/auth/register", func(c *gin.Context) {
		c.JSON(400, gin.H{"error": "桌面版无需注册"})
	})
	api.POST("/auth/reset-password", func(c *gin.Context) {
		c.JSON(400, gin.H{"error": "桌面版无需密码"})
	})

	authd := api.Group("")
	authd.Use(middleware.JWTAuth(jwtManager))

	authd.GET("/user/me", userHandler.Me)
	authd.PATCH("/user/me", userHandler.UpdateProfile)
	authd.GET("/user/usage", userHandler.UsageStats)

	authd.GET("/projects", projectHandler.ListProjects)
	authd.POST("/projects", projectHandler.CreateProject)
	authd.POST("/projects/import", projectHandler.ImportBook)
	authd.POST("/projects/import/batch", projectHandler.ImportBookBatch)
	authd.GET("/projects/search", projectHandler.SearchProjects)
	authd.GET("/projects/:id/search", projectHandler.SearchDocuments)
	authd.GET("/projects/:id", projectHandler.GetProject)
	authd.PATCH("/projects/:id", projectHandler.UpdateProject)
	authd.DELETE("/projects/:id", projectHandler.DeleteProject)
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
	authd.DELETE("/documents/:id", projectHandler.DeleteDocument)

	// 文档更新：镜像 cmd/server 的记忆提取包装（桌面版仅提取人物，章节摘要走联网 AI 时再补）
	authd.PATCH("/documents/:id", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "read body failed"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		projectHandler.UpdateDocument(c)
		if c.Writer.Status() == 200 {
			var req struct {
				Content string `json:"content"`
			}
			if json.Unmarshal(body, &req) != nil || req.Content == "" {
				return
			}
			var doc model.Document
			if err := db.First(&doc, c.Param("id")).Error; err != nil {
				return
			}
			uid := localUID
			if v, ok := c.Get("userID"); ok {
				if s, ok2 := v.(string); ok2 {
					uid = s
				}
			}
			if doc.Type == "characters" {
				content := req.Content
				pid := fmt.Sprintf("%d", doc.ProjectID)
				go memory.ExtractCharacters(context.Background(), aiClient, memStorage, uid, pid, content)
			}
		}
	})

	authd.GET("/trash", projectHandler.TrashListHandler)
	authd.POST("/trash/restore", projectHandler.RestoreDocument)
	authd.DELETE("/trash", projectHandler.EmptyTrash)
	authd.POST("/trash/destroy", projectHandler.DestroyDocument)

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
	authd.GET("/memory/relationships", memHandler.GetRelationships)
	authd.PUT("/memory/relationships", memHandler.PutRelationships)
	authd.GET("/memory/canvas", memHandler.GetCanvas)
	authd.PUT("/memory/canvas", memHandler.PutCanvas)
	authd.GET("/memory/:project_id", memHandler.GetProjectMemory)
	authd.PUT("/memory/:project_id/:file", memHandler.UpdateMemoryFile)
	authd.POST("/memory/:project_id/extract", memHandler.ExtractMemory)
	authd.POST("/memory/:project_id/distill", memHandler.DistillMemory)

	aiGroup := authd.Group("")
	aiGroup.Use(gw.RateLimitMiddleware())
	aiGroup.POST("/ai/generate", gw.HandleGenerate)
	aiGroup.POST("/ai/chat", gw.HandleChat)
	aiGroup.GET("/ai/models", gw.HandleGetModels)

	authd.GET("/tts/voices", tts.HandleVoices)
	authd.POST("/tts", tts.HandleTTS)

	buddy.RegisterRoutes(authd)
	buddy.SetDB(db) // 桌面版同款注入：writing_sessions 落库

	// 拼字房（多人拼字，纯内存无外部依赖，桌面版同样可用）
	sprint.RegisterRoutes(authd)

	// 便签/灵感随手记 + AI 味自检（右栏工作台 C 方案）
	sticky.SetDB(db)
	sticky.RegisterRoutes(authd)
	ai.RegisterSelfCheck(authd, aiClient, gateway.DefaultModels[0].ModelID)

	authd.GET("/projects/:id/stats", projectHandler.WritingStats)
	authd.GET("/projects/:id/threads", projectHandler.ListThreads)
	authd.POST("/projects/:id/threads", projectHandler.CreateThread)
	authd.PATCH("/threads/:id", projectHandler.UpdateThread)
	authd.DELETE("/threads/:id", projectHandler.DeleteThread)

	// 静态前端：与 exe 同目录的 web/（打包时放进安装包）
	webRoot := filepath.Join(filepath.Dir(exePath), "web")
	if st, err := os.Stat(webRoot); err == nil && st.IsDir() {
		r.NoRoute(func(c *gin.Context) {
			p := filepath.Join(webRoot, filepath.Clean("/"+c.Request.URL.Path))
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				c.File(p)
				return
			}
			c.File(filepath.Join(webRoot, "index.html"))
		})
	}

	// 随机空闲端口 + 自动开浏览器（双击 exe 的完整闭环）
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("监听端口失败:", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	log.Println("🚀 写作菌桌面版已启动:", url)
	log.Println("   数据目录:", dataRoot)
	go func() {
		time.Sleep(600 * time.Millisecond)
		openBrowser(url)
	}()
	if err := r.RunListener(listener); err != nil {
		log.Fatal("server failed:", err)
	}
}

// seedLocalUser 首次启动建本地作者，后续复用
func seedLocalUser(db *gorm.DB) string {
	const localEmail = "local@desktop.xiezuojun"
	var idn model.AuthIdentity
	if err := db.Where("provider = ? AND identifier = ?", "email", localEmail).First(&idn).Error; err == nil {
		return idn.UserID
	}
	uid := uuid.NewString()
	hash, _ := bcrypt.GenerateFromPassword([]byte("desktop"), bcrypt.MinCost)
	db.Create(&model.User{UserID: uid, Email: localEmail, Nickname: "我", PasswordHash: string(hash), Status: 1})
	db.Create(&model.AuthIdentity{UserID: uid, Provider: "email", Identifier: localEmail, CredentialHash: string(hash)})
	return uid
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
