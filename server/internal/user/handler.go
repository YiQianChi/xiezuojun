package user

import (
	crand "crypto/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	wxAppID    string
	wxAppSecret string
}

func NewHandler(db *gorm.DB, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		db:          db,
		jwtManager:  jwtManager,
		wxAppID:     os.Getenv("WX_MP_APPID"),
		wxAppSecret: os.Getenv("WX_MP_APPSECRET"),
	}
}

// ============== 注册/登录 ==============

type SendEmailCodeReq struct {
	Email string `json:"email" binding:"required,email"`
	Scene string `json:"scene" binding:"required"`
}

// SendEmailCode 发送邮箱验证码
func (h *Handler) SendEmailCode(c *gin.Context) {
	var req SendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Scene != "register" && req.Scene != "login" && req.Scene != "reset_password" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scene"})
		return
	}

	var existing model.AuthIdentity
	err := h.db.Where("provider = ? AND identifier = ?", "email", req.Email).First(&existing).Error
	if err != nil {
		err = h.db.Where("email = ?", req.Email).First(&model.User{}).Error
	}
	if req.Scene == "register" && err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if req.Scene == "login" && err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not registered"})
		return
	}
	if req.Scene == "reset_password" && err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not registered"})
		return
	}

	// 60 秒内同一邮箱同一场景只能发一次（Redis SET NX EX）
	ok, err := acquireSendSlot(req.Scene, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service unavailable"})
		return
	}
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "send too frequently"})
		return
	}

	code, err := generateEmailCode()
	if err != nil {
		releaseSendSlot(req.Scene, req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
		return
	}
	if err := storeCode(req.Scene, req.Email, hashVerificationCode("email", req.Email, req.Scene, code)); err != nil {
		releaseSendSlot(req.Scene, req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save code"})
		return
	}
	if err := sendEmailCode(req.Email, req.Scene, code); err != nil {
		releaseSendSlot(req.Scene, req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sent", "data": gin.H{"expires_in": 600}})
}

type RegisterReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Code     string `json:"code" binding:"required,len=6"`
}


// randomPenName 随机笔名：形容词+名词+数字，注册默认昵称——防邮箱直接当昵称在搭子等场景泄露
var penAdjs = []string{"墨水", "熬夜", "故事", "灵感", "断更", "催更", "柠檬", "月亮", "布谷", "青苔", "拾光", "听雨", "衔月", "踏雪", "执笔", "小饿", "打盹", "云朵", "萤火", "薄荷"}
var penNouns = []string{"菌"}

func randomPenName() string {
	// crypto 级随机，防可预测
	var b [2]byte
	_, _ = crand.Read(b[:])
	suffix := int(b[0])%90 + 10
	return penAdjs[int(b[0])%len(penAdjs)] + penNouns[int(b[1])%len(penNouns)] + strconv.Itoa(suffix)
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)

	// 检查邮箱是否已存在
	if h.emailIdentityExists(req.Email) {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if !h.consumeVerificationCode("email", req.Email, "register", req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired code"})
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
		Nickname:     randomPenName(),
		Plan:         "free",
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	verifiedAt := time.Now()
	_ = h.db.Create(&model.AuthIdentity{
		UserID:         user.UserID,
		Provider:       "email",
		Identifier:     req.Email,
		CredentialHash: string(hash),
		VerifiedAt:     &verifiedAt,
	}).Error

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
	Password string `json:"password"`
	Code     string `json:"code"`
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)

	var user model.User
	var identity model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "email", req.Email).First(&identity).Error; err == nil {
		if err := h.db.Where("user_id = ? AND status = 1", identity.UserID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
	} else {
		// 兼容旧 users.email 账号；登录成功后会补 identity
		if err := h.db.Where("email = ? AND status = 1", req.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
	}

	if req.Code != "" {
		if !h.consumeVerificationCode("email", req.Email, "login", req.Code) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired code"})
			return
		}
	} else {
		if req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password or code required"})
			return
		}
		passwordHash := identity.CredentialHash
		if passwordHash == "" {
			passwordHash = user.PasswordHash
		}
		if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
	}
	if identity.ID == 0 {
		verifiedAt := time.Now()
		_ = h.db.Create(&model.AuthIdentity{
			UserID:         user.UserID,
			Provider:       "email",
			Identifier:     req.Email,
			CredentialHash: user.PasswordHash,
			VerifiedAt:     &verifiedAt,
		}).Error
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

func (h *Handler) emailIdentityExists(email string) bool {
	var identity model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "email", email).First(&identity).Error; err == nil {
		return true
	}
	var user model.User
	return h.db.Where("email = ?", email).First(&user).Error == nil
}

func (h *Handler) consumeVerificationCode(targetType, target, scene, code string) bool {
	hash, ok := getCode(scene, target)
	if !ok {
		return false
	}
	if !verifyVerificationCodeHash(targetType, target, scene, code, hash) {
		// 输错计数，超限作废验证码防爆破
		if n, err := incrTry(scene, target); err == nil && n >= maxCodeTries {
			burnCode(scene, target)
		}
		return false
	}
	burnCode(scene, target)
	return true
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
	needsPenName := user.Nickname == "" || strings.Contains(user.Nickname, "@")
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id": user.ID, "user_id": user.UserID, "email": user.Email, "phone": user.Phone,
		"nickname": user.Nickname, "avatar": user.Avatar, "plan": user.Plan,
		"status": user.Status, "is_admin": user.IsAdmin,
		"created_at": user.CreatedAt, "updated_at": user.UpdatedAt,
		"needs_pen_name": needsPenName,
	}})
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
		if len([]rune(req.Nickname)) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称过长"})
			return
		}
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

// ChangePassword 已登录用户改密码（旧密码验证，无需验证码）
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：新密码至少 6 位"})
		return
	}
	var u model.User
	if err := h.db.Where("user_id = ?", userID).First(&u).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// 旧密码校验：identity 优先，回退 users.password_hash
	var identity model.AuthIdentity
	storedHash := u.PasswordHash
	if err := h.db.Where("user_id = ? AND provider = 'email'", u.UserID).First(&identity).Error; err == nil && identity.CredentialHash != "" {
		storedHash = identity.CredentialHash
	}
	if storedHash == "" || bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.OldPassword)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码不正确"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash"})
		return
	}
	h.db.Model(&model.User{}).Where("user_id = ?", u.UserID).Update("password_hash", string(hash))
	if identity.ID > 0 {
		h.db.Model(&identity).Update("credential_hash", string(hash))
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
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

// ResetPassword 重置密码（验证码 + 新密码）
func (h *Handler) ResetPassword(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Code     string `json:"code" binding:"required,len=6"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)

	if !h.consumeVerificationCode("email", req.Email, "reset_password", req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired code"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 更新 users.password_hash + identity.credential_hash
	h.db.Model(&model.User{}).Where("email = ?", req.Email).Update("password_hash", string(hash))
	var identity model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "email", req.Email).First(&identity).Error; err == nil {
		h.db.Model(&identity).Update("credential_hash", string(hash))
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset"})
}
