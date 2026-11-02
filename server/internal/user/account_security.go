package user

// 账号与安全：绑定状态查询、设置邮箱、绑定手机号、微信扫码绑定（ticket 机制）。
// 供 ProfileModal「账号与安全」弹窗使用。

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

func curUserID(c *gin.Context) string {
	v, _ := c.Get("userID")
	s, _ := v.(string)
	return s
}

// SecurityInfo 当前账号的绑定状态（扩展进 /user/me 响应）
func securityInfo(db *gorm.DB, user *model.User) gin.H {
	var wx int64
	db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'wechat'", user.UserID).Count(&wx)
	var email int64
	db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'email'", user.UserID).Count(&email)
	hasPwd := user.PasswordHash != ""
	if !hasPwd {
		var id model.AuthIdentity
		if db.Where("user_id = ? AND provider = 'email'", user.UserID).First(&id).Error == nil {
			hasPwd = id.CredentialHash != ""
		}
	}
	return gin.H{
		"email_bound":  user.Email != "" || email > 0,
		"phone_bound":  user.Phone != "",
		"wechat_bound": wx > 0,
		"has_password": hasPwd,
	}
}

// SetEmail 给当前账号设置邮箱+密码（邮箱必须未被注册）
func (h *Handler) SetEmail(c *gin.Context) {
	uid := curUserID(c)
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少 6 位"})
		return
	}
	// 邮箱归属查重：已被别的账号占用则拒绝
	var idn model.AuthIdentity
	byIdentity := h.db.Where("provider = ? AND identifier = ?", "email", req.Email).First(&idn).Error == nil
	var other model.User
	byUser := h.db.Where("email = ?", req.Email).First(&other).Error == nil
	if (byIdentity && idn.UserID != uid) || (byUser && other.UserID != uid) {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被其他账号使用"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.User{}).Where("user_id = ?", uid).Updates(map[string]interface{}{
			"email": req.Email, "password_hash": string(hash),
		}).Error; e != nil {
			return e
		}
		if byIdentity && idn.UserID == uid {
			return tx.Model(&model.AuthIdentity{}).Where("id = ?", idn.ID).
				Update("credential_hash", string(hash)).Error
		}
		verifiedAt := time.Now()
		return tx.Create(&model.AuthIdentity{
			UserID: uid, Provider: "email", Identifier: req.Email,
			CredentialHash: string(hash), VerifiedAt: &verifiedAt,
		}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// BindPhone 给当前账号绑定手机号（短信验证码，scene=bind）
func (h *Handler) BindPhone(c *gin.Context) {
	uid := curUserID(c)
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone and code required"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if !phoneRe.MatchString(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone"})
		return
	}
	if !h.consumeVerificationCode("phone", req.Phone, "bind", req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired code"})
		return
	}
	// 手机号归属查重
	var idn model.AuthIdentity
	if h.db.Where("provider = ? AND identifier = ?", "phone", req.Phone).First(&idn).Error == nil && idn.UserID != uid {
		c.JSON(http.StatusConflict, gin.H{"error": "该手机号已被其他账号使用"})
		return
	}
	var other model.User
	if h.db.Where("phone = ? AND status = 1", req.Phone).First(&other).Error == nil && other.UserID != uid {
		c.JSON(http.StatusConflict, gin.H{"error": "该手机号已被其他账号使用"})
		return
	}
	var user model.User
	if err := h.db.Where("user_id = ?", uid).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.User{}).Where("user_id = ?", uid).Update("phone", req.Phone).Error; e != nil {
			return e
		}
		ensurePhoneIdentity(tx, uid, req.Phone)
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// WxBindTicket 领取一次性微信绑定票据（5 分钟），拼进扫码 state 里带回
func (h *Handler) WxBindTicket(c *gin.Context) {
	uid := curUserID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ticket := uuid.NewString()
	vcodeRdb.Set(vcodeCtx, "xzj:wxbind:"+ticket, uid, 5*time.Minute)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ticket": ticket}})
}

// bindWechatToUser 把微信身份挂到指定用户；身份属于孤儿号则合并迁作品，属于"有家室"的号则 409
func (h *Handler) bindWechatToUser(uid, identifier string) error {
	var idn model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "wechat", identifier).First(&idn).Error; err == nil {
		if idn.UserID == uid {
			return nil // 已绑定
		}
		var other int64
		h.db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider <> 'wechat'", idn.UserID).Count(&other)
		var orphan model.User
		h.db.Where("user_id = ?", idn.UserID).First(&orphan)
		if other > 0 || orphan.IsAdmin {
			return errAccountConflict
		}
		return h.db.Transaction(func(tx *gorm.DB) error {
			if e := tx.Model(&model.Project{}).Where("user_id = ?", idn.UserID).Update("user_id", uid).Error; e != nil {
				return e
			}
			if e := tx.Model(&model.AuthIdentity{}).Where("id = ?", idn.ID).Update("user_id", uid).Error; e != nil {
				return e
			}
			return tx.Model(&model.User{}).Where("user_id = ?", idn.UserID).Update("status", 0).Error
		})
	}
	verifiedAt := time.Now()
	return h.db.Create(&model.AuthIdentity{
		UserID: uid, Provider: "wechat", Identifier: identifier, VerifiedAt: &verifiedAt,
	}).Error
}

var errPhoneTaken = errors.New("phone taken") // 占位：预留给换绑手机场景
