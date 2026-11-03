package user

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"xiezuojun-server/internal/model"
)

// ===== 统一身份冲突处理（规则拍板 2026-11-03，对齐大厂惯例） =====
// 绑定微信/手机遇「已被其他账号占用」：
//   - 对方还有其他登录方式 → 验证拥有对方账号（对方手机短信 / 邮箱验证码，scene=rebind）→ 改绑到当前账号
//   - 对方只剩这一个身份（孤儿号）→ 不能改绑（会变无人能登录的号）→ 对方账号并入当前账号（作品随迁，对方禁用）
// 邮箱被占：不做合并/改绑，提示换邮箱（BindEmail 场景）。

const idConflictTicketTTL = 10 * time.Minute

type idConflictTicket struct {
	Provider   string // wechat | phone
	Identifier string
	CurUID     string
	OtherUID   string
	VerifyType string // phone | email（孤儿号为空）
	VerifyTo   string // 验证码发到哪
}

func maskHint(s string) string {
	if strings.Contains(s, "@") {
		parts := strings.SplitN(s, "@", 2)
		n := parts[0]
		if len(n) > 2 {
			n = n[:2] + "***"
		}
		return n + "@" + parts[1]
	}
	if len(s) >= 7 {
		return s[:3] + "****" + s[len(s)-4:]
	}
	return "***"
}

// createIdConflictTicket 校验冲突并落 redis 票据；返回给前端的冲突信息
func (h *Handler) createIdConflictTicket(uid, provider, identifier string) (gin.H, error) {
	if provider != "wechat" && provider != "phone" {
		return nil, errors.New("unsupported provider")
	}
	var idn model.AuthIdentity
	if err := h.db.Where("provider = ? AND identifier = ?", provider, identifier).First(&idn).Error; err != nil {
		return nil, errors.New("该" + map[string]string{"wechat": "微信", "phone": "手机号"}[provider] + "未被占用，可直接绑定")
	}
	if idn.UserID == uid {
		return nil, errors.New("该身份已绑在当前账号上")
	}
	var other model.User
	if err := h.db.Where("user_id = ? AND status = 1", idn.UserID).First(&other).Error; err != nil {
		return nil, errors.New("对方账号已禁用，请联系客服")
	}

	// 对方的可用验证渠道与是否孤儿
	var otherIDns []model.AuthIdentity
	h.db.Where("user_id = ? AND provider <> ?", other.UserID, provider).Find(&otherIDns)
	verifyType, verifyTo := "", ""
	for _, id := range otherIDns {
		if id.Provider == "phone" && verifyType == "" {
			verifyType, verifyTo = "phone", id.Identifier
		}
		if id.Provider == "email" && verifyType != "phone" {
			verifyType, verifyTo = "email", id.Identifier
		}
	}
	if verifyType == "" && other.Phone != "" {
		verifyType, verifyTo = "phone", other.Phone
	}
	if verifyType == "" && other.Email != "" {
		verifyType, verifyTo = "email", other.Email
	}
	orphan := verifyType == ""

	// 建票据
	ticket := uuid.NewString()
	t := idConflictTicket{Provider: provider, Identifier: identifier, CurUID: uid, OtherUID: other.UserID, VerifyType: verifyType, VerifyTo: verifyTo}
	if err := vcodeRdb.Set(vcodeCtx, "idconf:"+ticket, marshalTicket(t), idConflictTicketTTL).Err(); err != nil {
		return nil, err
	}
	hint := "其他账号"
	if verifyTo != "" {
		hint = maskHint(verifyTo)
	}
	return gin.H{
		"ticket": ticket, "provider": provider,
		"other_hint": hint, "verify_type": verifyType, "orphan": orphan,
	}, nil
}

func marshalTicket(t idConflictTicket) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s", t.Provider, t.Identifier, t.CurUID, t.OtherUID, t.VerifyType, t.VerifyTo)
}
func unmarshalTicket(s string) idConflictTicket {
	p := strings.Split(s, "|")
	for len(p) < 6 {
		p = append(p, "")
	}
	return idConflictTicket{p[0], p[1], p[2], p[3], p[4], p[5]}
}
func loadIdConflictTicket(ticket string) (idConflictTicket, error) {
	s, err := vcodeRdb.Get(vcodeCtx, "idconf:"+ticket).Result()
	if err == redis.Nil {
		return idConflictTicket{}, errors.New("冲突会话已过期，请重新发起")
	}
	if err != nil {
		return idConflictTicket{}, err
	}
	return unmarshalTicket(s), nil
}

// CreateIdentityConflict POST /user/identity-conflict —— 手机绑定 409 后领冲突票据
func (h *Handler) CreateIdentityConflict(c *gin.Context) {
	uid := curUserID(c)
	var req struct {
		Provider   string `json:"provider" binding:"required"`
		Identifier string `json:"identifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and identifier required"})
		return
	}
	// 领票限速：单用户 10 次/小时（防拿别人手机号刷短信）
	key := "idconfcnt:" + uid
	if n, _ := vcodeRdb.Incr(vcodeCtx, key).Result(); n == 1 {
		vcodeRdb.Expire(vcodeCtx, key, time.Hour)
	} else if n > 10 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作过于频繁，请 1 小时后再试"})
		return
	}
	payload, err := h.createIdConflictTicket(uid, req.Provider, req.Identifier)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// SendConflictCode POST /user/identity-conflict/send-code —— 验证码发给对方账号的渠道
func (h *Handler) SendConflictCode(c *gin.Context) {
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket required"})
		return
	}
	t, err := loadIdConflictTicket(req.Ticket)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.VerifyType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "对方账号没有可验证的渠道"})
		return
	}
	code, err := generateEmailCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
		return
	}
	if err := storeCode("rebind", t.VerifyTo, hashVerificationCode(t.VerifyType, t.VerifyTo, "rebind", code)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save code"})
		return
	}
	if t.VerifyType == "phone" {
		if err := sendSmsCode(t.VerifyTo, code, "rebind"); err != nil {
			log.Printf("[idconf] sms send fail to=%s err=%v", maskHint(t.VerifyTo), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send sms"})
			return
		}
	} else {
		if err := sendEmailCode(t.VerifyTo, "rebind", code); err != nil {
			log.Printf("[idconf] email send fail to=%s err=%v", maskHint(t.VerifyTo), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "sent", "data": gin.H{"expires_in": 600, "sent_to": maskHint(t.VerifyTo)}})
}

// ResolveIdentityConflict POST /user/identity-conflict/resolve —— 改绑：身份从对方迁到当前账号
func (h *Handler) ResolveIdentityConflict(c *gin.Context) {
	uid := curUserID(c)
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket and code required"})
		return
	}
	t, err := loadIdConflictTicket(req.Ticket)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.CurUID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权处理该冲突"})
		return
	}
	if t.VerifyType == "" || !h.consumeVerificationCode(t.VerifyType, t.VerifyTo, "rebind", strings.TrimSpace(req.Code)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误或已过期"})
		return
	}
	var otherEmail string
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 身份归属复核（票据 10 分钟内可能已变）
		var idn model.AuthIdentity
		if err := tx.Where("provider = ? AND identifier = ?", t.Provider, t.Identifier).First(&idn).Error; err != nil {
			return errors.New("该身份已不在对方账号上")
		}
		if idn.UserID != t.OtherUID {
			return errors.New("该身份归属已变化，请重新发起")
		}
		if err := tx.Model(&model.AuthIdentity{}).Where("id = ?", idn.ID).Update("user_id", t.CurUID).Error; err != nil {
			return err
		}
		var other model.User
		if err := tx.Where("user_id = ?", t.OtherUID).First(&other).Error; err == nil {
			otherEmail = other.Email
			if t.Provider == "phone" && other.Phone == t.Identifier {
				tx.Model(&model.User{}).Where("user_id = ?", other.UserID).Update("phone", "")
			}
		}
		if t.Provider == "phone" {
			if err := tx.Model(&model.User{}).Where("user_id = ?", t.CurUID).Update("phone", t.Identifier).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	vcodeRdb.Del(vcodeCtx, "idconf:"+req.Ticket)
	// 通知对方账号（能通知到就通知，改绑对对方是高感事件）
	go func() {
		if otherEmail == "" {
			return
		}
		cfg := loadMailConfig()
		if cfg.Host == "" || cfg.Username == "" {
			return
		}
		what := "微信"
		if t.Provider == "phone" {
			what = "手机号"
		}
		plain := "你的写作菌账号绑定的" + what + "刚刚改绑到了另一个账号。\n\n若非本人操作，请立即登录写作菌修改密码并检查账号安全。\n\n—— 写作菌"
		html := "<div style=\"max-width:420px;margin:0 auto;font-family:-apple-system,'PingFang SC',sans-serif;padding:24px\"><div style=\"font-size:15px;line-height:1.8\">你的写作菌账号绑定的<b>" + what + "</b>刚刚改绑到了另一个账号。<br><br>若非本人操作，请立即登录写作菌修改密码并检查账号安全。</div></div>"
		if err := sendMailRaw(cfg, otherEmail, "写作菌账号改绑通知", plain, html); err != nil {
			log.Printf("[idconf] notify mail fail to=%s err=%v", maskHint(otherEmail), err)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "已改绑到当前账号"})
}

// MergeIdentityConflict POST /user/identity-conflict/merge —— 孤儿号并入当前账号（作品随迁，对方禁用）
func (h *Handler) MergeIdentityConflict(c *gin.Context) {
	uid := curUserID(c)
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket required"})
		return
	}
	t, err := loadIdConflictTicket(req.Ticket)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.CurUID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权处理该冲突"})
		return
	}
	if t.VerifyType != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "对方账号还有其他登录方式，走验证改绑即可"})
		return
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var idn model.AuthIdentity
		if err := tx.Where("provider = ? AND identifier = ?", t.Provider, t.Identifier).First(&idn).Error; err != nil {
			return errors.New("该身份已不在对方账号上")
		}
		if idn.UserID != t.OtherUID {
			return errors.New("该身份归属已变化，请重新发起")
		}
		// 对方作品随迁到当前账号
		if err := tx.Model(&model.Project{}).Where("user_id = ?", t.OtherUID).Update("user_id", t.CurUID).Error; err != nil {
			return err
		}
		// 冲突身份挂到当前账号
		if err := tx.Model(&model.AuthIdentity{}).Where("id = ?", idn.ID).Update("user_id", t.CurUID).Error; err != nil {
			return err
		}
		if t.Provider == "phone" {
			if err := tx.Model(&model.User{}).Where("user_id = ?", t.CurUID).Update("phone", t.Identifier).Error; err != nil {
				return err
			}
		}
		// 孤儿号禁用
		return tx.Model(&model.User{}).Where("user_id = ?", t.OtherUID).Update("status", 0).Error
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	vcodeRdb.Del(vcodeCtx, "idconf:"+req.Ticket)
	c.JSON(http.StatusOK, gin.H{"message": "已合并：对方账号的作品已并入当前账号"})
}
