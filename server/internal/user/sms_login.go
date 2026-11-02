package user

// 手机号验证码登录（登录即注册）。验证码存储/限频/防爆破复用邮箱那套（vcode_redis.go）。

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

type SendSmsCodeReq struct {
	Phone         string `json:"phone" binding:"required"`
	Scene         string `json:"scene" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

// SendSmsCode 发送短信验证码。scene 目前只支持 login（验证码登录即注册，无需注册场景）
func (h *Handler) SendSmsCode(c *gin.Context) {
	if !loadSmsConfig().configured() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "sms not configured"})
		return
	}
	var req SendSmsCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if !phoneRe.MatchString(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone"})
		return
	}
	if req.Scene != "login" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scene"})
		return
	}

	// 渐进式验证码：当日该号/IP 已发 >=2 条时，必须先过数学验证码（正常用户无感）
	if smsNeedsCaptcha(req.Phone, c.ClientIP()) {
		if !verifyCaptcha(req.CaptchaID, req.CaptchaAnswer) {
			id, q := newCaptcha()
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error":   "captcha required",
				"captcha": gin.H{"id": id, "q": q},
			})
			return
		}
	}

	ok, err := acquireSendSlot(req.Scene, req.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service unavailable"})
		return
	}
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "send too frequently"})
		return
	}
	// 防刷配额：单号 5/时 10/日、单IP 10/时 30/日、全站日熔断（SMS_DAILY_CAP）
	if acquireSmsQuota(req.Phone, c.ClientIP()) != "" {
		releaseSendSlot(req.Scene, req.Phone)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "send limit exceeded"})
		return
	}
	code, err := generateEmailCode() // 名字带 email 而已，就是通用 6 位数字码
	if err != nil {
		releaseSendSlot(req.Scene, req.Phone)
		releaseSmsQuota(req.Phone, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
		return
	}
	if err := storeCode(req.Scene, req.Phone, hashVerificationCode("phone", req.Phone, req.Scene, code)); err != nil {
		releaseSendSlot(req.Scene, req.Phone)
		releaseSmsQuota(req.Phone, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save code"})
		return
	}
	if err := sendSmsCode(req.Phone, code, req.Scene); err != nil {
		log.Printf("[sms] send failed phone=%s scene=%s err=%v", req.Phone, req.Scene, err)
		releaseSendSlot(req.Scene, req.Phone)
		releaseSmsQuota(req.Phone, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send sms"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sent", "data": gin.H{"expires_in": 600}})
}

type SmsLoginReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required,len=6"`
}

// SmsLogin 手机号验证码登录：无账号则自动建号
func (h *Handler) SmsLogin(c *gin.Context) {
	var req SmsLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if !phoneRe.MatchString(phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone"})
		return
	}
	if !h.consumeVerificationCode("phone", phone, "login", req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired code"})
		return
	}

	user, isNew, err := h.findOrCreatePhoneUser(phone)
	if err != nil {
		if err.Error() == "account disabled" {
			c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	access, refresh, err := h.issueTokens(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user":          loginUserPayload(user),
			"access_token":  access,
			"refresh_token": refresh,
			"is_new":        isNew,
		},
	})
}

// findOrCreatePhoneUser 按手机号查号/建号
func (h *Handler) findOrCreatePhoneUser(phone string) (*model.User, bool, error) {
	var identity model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "phone", phone).First(&identity).Error; err == nil {
		var user model.User
		if err := h.db.Where("user_id = ? AND status = 1", identity.UserID).First(&user).Error; err != nil {
			return nil, false, errors.New("account disabled")
		}
		return &user, false, nil
	}
	// users.phone 兜底（微信绑定流程写主表但 identity 缺失的旧数据）
	var user model.User
	if err := h.db.Where("phone = ? AND status = 1", phone).First(&user).Error; err == nil {
		ensurePhoneIdentity(h.db, user.UserID, phone)
		return &user, false, nil
	}
	user = model.User{
		UserID:   uuid.NewString(),
		Phone:    phone,
		Nickname: randomPenName(),
		Plan:     "free",
		Status:   1,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return nil, false, fmt.Errorf("failed to create user")
	}
	ensurePhoneIdentity(h.db, user.UserID, phone)
	return &user, true, nil
}

// issueTokens 生成 JWT 对并落 refresh_tokens（sms/wx-phone 登录共用）
func (h *Handler) issueTokens(c *gin.Context, user *model.User) (string, string, error) {
	access, refresh, err := h.jwtManager.GeneratePair(user.UserID, user.Email, user.Plan)
	if err != nil {
		return "", "", err
	}
	h.db.Create(&model.RefreshToken{
		UserID:    user.UserID,
		Token:     refresh,
		DeviceID:  c.GetHeader("X-Device-ID"),
		IP:        c.ClientIP(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	return access, refresh, nil
}

func loginUserPayload(user *model.User) gin.H {
	return gin.H{
		"id":       user.ID,
		"user_id":  user.UserID,
		"email":    user.Email,
		"phone":    user.Phone,
		"nickname": user.Nickname,
		"plan":     user.Plan,
	}
}

// ensurePhoneIdentity 补建 phone identity（已存在则跳过）；可传事务
func ensurePhoneIdentity(db *gorm.DB, userID, phone string) {
	var cnt int64
	db.Model(&model.AuthIdentity{}).Where("provider = ? AND identifier = ?", "phone", phone).Count(&cnt)
	if cnt == 0 {
		verifiedAt := time.Now()
		db.Create(&model.AuthIdentity{
			UserID:     userID,
			Provider:   "phone",
			Identifier: phone,
			VerifiedAt: &verifiedAt,
		})
	}
}
