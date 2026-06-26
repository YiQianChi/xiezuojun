package user

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"xiezuojun-server/internal/auth"
	"xiezuojun-server/internal/model"
)

// Handler 用户相关 API
type Handler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
}

func NewHandler(db *gorm.DB, jwtManager *auth.JWTManager) *Handler {
	return &Handler{db: db, jwtManager: jwtManager}
}

// ============== 注册/登录 ==============

type RegisterReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查邮箱是否已存在
	var existing model.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := model.User{
		UserID:       uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Nickname:     req.Email,
		Plan:         "free",
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.jwtManager.GeneratePair(user.UserID, user.Email, user.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 保存 refresh token
	rt := model.RefreshToken{
		UserID:    user.UserID,
		Token:     refresh,
		DeviceID:  c.GetHeader("X-Device-ID"),
		IP:        c.ClientIP(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	h.db.Create(&rt)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user": gin.H{
				"id":      user.ID,
				"user_id": user.UserID,
				"email":   user.Email,
				"nickname": user.Nickname,
				"plan":    user.Plan,
			},
			"access_token":  access,
			"refresh_token": refresh,
		},
	})
}

type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user model.User
	if err := h.db.Where("email = ? AND status = 1", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	access, refresh, err := h.jwtManager.GeneratePair(user.UserID, user.Email, user.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	rt := model.RefreshToken{
		UserID:    user.UserID,
		Token:     refresh,
		DeviceID:  c.GetHeader("X-Device-ID"),
		IP:        c.ClientIP(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	h.db.Create(&rt)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user": gin.H{
				"id":      user.ID,
				"user_id": user.UserID,
				"email":   user.Email,
				"nickname": user.Nickname,
				"plan":    user.Plan,
				"avatar":  user.Avatar,
			},
			"access_token":  access,
			"refresh_token": refresh,
		},
	})
}

// Refresh 刷新 access token
func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 refresh token 签名
	if err := h.jwtManager.ParseRefresh(req.RefreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// 查数据库确认 token 存在且未过期
	var rt model.RefreshToken
	if err := h.db.Where("token = ? AND expires_at > ?", req.RefreshToken, time.Now()).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired or revoked"})
		return
	}

	var user model.User
	if err := h.db.Where("user_id = ?", rt.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	access, newRefresh, err := h.jwtManager.GeneratePair(user.UserID, user.Email, user.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 更新 refresh token（轮换）
	h.db.Where("id = ?", rt.ID).Delete(&model.RefreshToken{})
	newRT := model.RefreshToken{
		UserID:    user.UserID,
		Token:     newRefresh,
		DeviceID:  rt.DeviceID,
		IP:        c.ClientIP(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	h.db.Create(&newRT)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"access_token":  access,
			"refresh_token": newRefresh,
		},
	})
}

// ============== 用户信息 ==============

// Me 获取当前用户信息
func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.Get("userID")
	var user model.User
	if err := h.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateProfile 更新用户资料
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}

	if err := h.db.Model(&model.User{}).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// UsageStats 获取用户用量统计
func (h *Handler) UsageStats(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(string)

	// 计算总字数（所有文档 word_count 之和）
	var totalWords int64
	h.db.Model(&model.Document{}).
		Joins("JOIN projects ON projects.id = documents.project_id").
		Where("projects.user_id = ? AND documents.status = 1", uid).
		Select("COALESCE(SUM(documents.word_count), 0)").
		Scan(&totalWords)

	// 计算本月字数（按 created_at 过滤）
	var monthWords int64
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	h.db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND created_at >= ?", uid, startOfMonth).
		Select("COALESCE(SUM(delta), 0)").
		Scan(&monthWords)

	// 今日字数
	var todayWords int64
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	h.db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND created_at >= ?", uid, startOfDay).
		Select("COALESCE(SUM(delta), 0)").
		Scan(&todayWords)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_words": totalWords,
			"month_words": monthWords,
			"today_words": todayWords,
		},
	})
}
