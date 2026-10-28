package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xiezuojun-server/internal/model"
)

// 微信小程序一键登录：wx.login code -> 后端 jscode2session -> openid 建号/登录 -> 发 JWT
// AppID/AppSecret 从环境变量 WX_MP_APPID / WX_MP_APPSECRET 读取（未配置时接口返回 501）。

type wechatSessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type WxLoginReq struct {
	Code string `json:"code" binding:"required"`
}

func (h *Handler) wxConfigured() bool {
	return h.wxAppID != "" && h.wxAppSecret != ""
}

// WxLogin 小程序微信登录
func (h *Handler) WxLogin(c *gin.Context) {
	if !h.wxConfigured() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "wechat login not configured"})
		return
	}
	var req WxLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code required"})
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code required"})
		return
	}

	sess, err := h.jscode2session(req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wechat login failed: " + err.Error()})
		return
	}

	identifier := sess.OpenID
	if identifier == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wechat login failed: empty openid"})
		return
	}
	// 优先 unionid（多端统一身份），拿不到退 openid
	if sess.UnionID != "" {
		identifier = sess.UnionID
	}

	var user model.User
	err = h.db.Where("user_id = ?", func() string { // 先查 identity
		var id model.AuthIdentity
		if e := h.db.Where("provider = ? AND identifier = ?", "wechat", identifier).First(&id).Error; e == nil {
			return id.UserID
		}
		return ""
	}()).First(&user).Error

	if err != nil {
		// 新用户：自动建号（昵称走随机笔名，邮箱空）
		user = model.User{
			UserID:   uuid.NewString(),
			Email:    "",
			Nickname: randomPenName(),
			Plan:     "free",
			Status:   1,
		}
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
		verifiedAt := time.Now()
		_ = h.db.Create(&model.AuthIdentity{
			UserID:     user.UserID,
			Provider:   "wechat",
			Identifier: identifier,
			VerifiedAt: &verifiedAt,
		}).Error
	} else {
		if user.Status != 1 {
			c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
			return
		}
		// 老用户但 identity 缺失（理论上不会，兜底补）
		var cnt int64
		h.db.Model(&model.AuthIdentity{}).Where("provider = ? AND identifier = ?", "wechat", identifier).Count(&cnt)
		if cnt == 0 {
			verifiedAt := time.Now()
			_ = h.db.Create(&model.AuthIdentity{
				UserID:     user.UserID,
				Provider:   "wechat",
				Identifier: identifier,
				VerifiedAt: &verifiedAt,
			}).Error
		}
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
			},
			"access_token":  access,
			"refresh_token": refresh,
			"is_new":        user.ID != 0 && err == nil && user.Email == "" && user.Nickname != "",
		},
	})
}

func (h *Handler) jscode2session(code string) (*wechatSessionResp, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		h.wxAppID, h.wxAppSecret, code,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out wechatSessionResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("weixin api errcode=%d errmsg=%s", out.ErrCode, out.ErrMsg)
	}
	return &out, nil
}
