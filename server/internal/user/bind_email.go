package user

import (
	"errors"
	"log"
	"net/http"
	"time"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

// BindEmail 把当前（微信扫码建出来的）账号绑定到已有邮箱账号：
// 校验邮箱+密码 → 微信身份和作品迁到邮箱账号 → 当前孤儿号禁用 → 发邮箱账号的 token
func (h *Handler) BindEmail(c *gin.Context) {
	userID, _ := c.Get("userID")
	curID, _ := userID.(string)
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// 校验邮箱账号 + 密码（与 Login 同一套：identity 优先，users 兜底）
	var target model.User
	var identity model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", "email", req.Email).First(&identity).Error; err == nil {
		if err := h.db.Where("user_id = ? AND status = 1", identity.UserID).First(&target).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
	} else if err := h.db.Where("email = ? AND status = 1", req.Email).First(&target).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	hash := identity.CredentialHash
	if hash == "" {
		hash = target.PasswordHash
	}
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		incrMergeFail(req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	if target.UserID == curID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "这就是当前账号"})
		return
	}

	// 防爆破：同一邮箱连续失败 5 次锁 1 小时（Redis 计数）
	if n, _ := vcodeRdb.Get(vcodeCtx, "mergefail:"+req.Email).Int64(); n >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "尝试次数过多，请 1 小时后再试"})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 目标账号已有微信身份 → 冲突，不能两个微信号挤一个号
		var cnt int64
		tx.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'wechat'", target.UserID).Count(&cnt)
		if cnt > 0 {
			return errAccountConflict
		}
		// 当前账号必须是纯微信孤儿号（除微信外无其他登录方式），否则不静默并号
		var other int64
		tx.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider <> 'wechat'", curID).Count(&other)
		if other > 0 {
			return errAccountConflict
		}
		var wxIdentities []model.AuthIdentity
		if err := tx.Where("user_id = ? AND provider = 'wechat'", curID).Find(&wxIdentities).Error; err != nil {
			return err
		}
		if len(wxIdentities) == 0 {
			return errors.New("当前账号没有微信身份可迁移")
		}
		// 作品随迁
		if err := tx.Model(&model.Project{}).Where("user_id = ?", curID).Update("user_id", target.UserID).Error; err != nil {
			return err
		}
		// 微信身份挂到邮箱账号
		if err := tx.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'wechat'", curID).Update("user_id", target.UserID).Error; err != nil {
			return err
		}
		// 孤儿号禁用
		return tx.Model(&model.User{}).Where("user_id = ?", curID).Update("status", 0).Error
	})
	if err != nil {
		if errors.Is(err, errAccountConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "账号冲突：目标邮箱已绑定其他微信，或当前账号已有其他登录方式"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.issueTokens(c, &target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成登录态失败"})
		return
	}
	// 合并成功：清失败计数 + 异步通知被并入的账号（密码泄露场景受害者能察觉）
	vcodeRdb.Del(vcodeCtx, "mergefail:"+req.Email)
	go func() {
		cfg := loadMailConfig()
		if cfg.Host == "" || cfg.Username == "" {
			return
		}
		plain := "你的写作菌账号刚刚与另一个账号合并，另一账号的作品已并入本账号。\n\n若非本人操作，请立即登录写作菌修改密码。\n\n—— 写作菌"
		html := "<div style=\"max-width:420px;margin:0 auto;font-family:-apple-system,'PingFang SC',sans-serif;padding:24px\">" +
			"<div style=\"font-size:15px;line-height:1.8\">你的写作菌账号刚刚与另一个账号<b>合并</b>，另一账号的作品已并入本账号。<br><br>若非本人操作，请立即登录写作菌修改密码。</div></div>"
		if err := sendMailRaw(cfg, req.Email, "写作菌账号合并通知", plain, html); err != nil {
			log.Printf("[bind-email] merge notify mail fail to=%s err=%v", req.Email, err)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"user":          loginUserPayload(&target),
		"access_token":  access,
		"refresh_token": refresh,
	}})
}

// incrMergeFail 合并试密码失败计数（Redis，5 次锁 1 小时，防爆破）
func incrMergeFail(email string) {
	key := "mergefail:" + email
	n, err := vcodeRdb.Incr(vcodeCtx, key).Result()
	if err == nil && n == 1 {
		vcodeRdb.Expire(vcodeCtx, key, time.Hour)
	}
}
