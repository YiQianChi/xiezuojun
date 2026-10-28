package user

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xiezuojun-server/internal/model"
)

// 微信 OAuth 网页授权（微信内置浏览器打开 xiezuojun.com 时的一键登录）。
// 与小程序 wx.login 是两条链路：这里走开放平台/公众号 OAuth2。
// 配置：WX_OAUTH_APPID / WX_OAUTH_SECRET（开放平台网站应用或服务号，审核通过后填入 .env 即点亮）。
// 回调地址：https://xiezuojun.com/api/auth/wx-oauth/callback（要在微信后台登记为授权回调域名）

func (h *Handler) oauthConfigured() bool {
	return h.wxOauthAppID != "" && h.wxOauthSecret != ""
}

// WxOauthStart 发起微信网页授权：跳微信授权页，用户同意后带 code 回调
func (h *Handler) WxOauthStart(c *gin.Context) {
	if !h.oauthConfigured() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "wechat oauth not configured"})
		return
	}
	redirect := c.Query("redirect")
	if redirect == "" || !strings.HasPrefix(redirect, "http") {
		redirect = "https://xiezuojun.com/login"
	}
	cb := "https://xiezuojun.com/api/auth/wx-oauth/callback"
	u := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
		h.wxOauthAppID, url.QueryEscape(cb), url.QueryEscape(redirect),
	)
	c.Redirect(http.StatusFound, u)
}

type oauthTokenResp struct {
	AccessToken  string `json:"access_token"`
	OpenID       string `json:"openid"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

// WxOauthCallback 微信授权回调：code 换 openid/unionid，建号/登录，token 桥回前端
func (h *Handler) WxOauthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := c.Query("state")
	front := "https://xiezuojun.com/login"
	if strings.HasPrefix(state, "http") {
		front = state
	}
	if !h.oauthConfigured() || code == "" {
		c.Redirect(http.StatusFound, front+"?wx_err=config")
		return
	}

	tok, err := h.oauthGetToken(code)
	if err != nil {
		c.Redirect(http.StatusFound, front+"?wx_err="+url.QueryEscape(err.Error()))
		return
	}
	identifier := tok.UnionID
	if identifier == "" {
		identifier = tok.OpenID
	}
	if identifier == "" {
		c.Redirect(http.StatusFound, front+"?wx_err=empty_openid")
		return
	}

	user, err := h.findOrCreateWechatUser(identifier)
	if err != nil {
		c.Redirect(http.StatusFound, front+"?wx_err="+url.QueryEscape(err.Error()))
		return
	}

	access, refresh, err := h.jwtManager.GeneratePair(user.UserID, user.Email, user.Plan)
	if err != nil {
		c.Redirect(http.StatusFound, front+"?wx_err=token")
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

	// token 桥回前端：前端 App.vue 会收 ?xzj_at=… 进 localStorage（复用小程序桥接同一套）
	sep := "?"
	if strings.Contains(front, "?") {
		sep = "&"
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s%sxzj_at=%s&xzj_rt=%s", front, sep, url.QueryEscape(access), url.QueryEscape(refresh)))
}

func (h *Handler) oauthGetToken(code string) (*oauthTokenResp, error) {
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		h.wxOauthAppID, h.wxOauthSecret, code,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out oauthTokenResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("weixin oauth errcode=%d", out.ErrCode)
	}
	return &out, nil
}

// findOrCreateWechatUser 微信身份查号/建号（小程序与 OAuth 共用，unionid 相同则同一账号）
func (h *Handler) findOrCreateWechatUser(identifier string) (*model.User, error) {
	var identity model.AuthIdentity
	var user model.User
	if err := h.db.Where("provider = ? AND identifier = ?", "wechat", identifier).First(&identity).Error; err == nil {
		if err := h.db.Where("user_id = ? AND status = 1", identity.UserID).First(&user).Error; err != nil {
			return nil, fmt.Errorf("account disabled")
		}
		return &user, nil
	}
	// 新用户建号
	user = model.User{
		UserID:   uuid.NewString(),
		Email:    "",
		Nickname: randomPenName(),
		Plan:     "free",
		Status:   1,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user")
	}
	verifiedAt := time.Now()
	_ = h.db.Create(&model.AuthIdentity{
		UserID:     user.UserID,
		Provider:   "wechat",
		Identifier: identifier,
		VerifiedAt: &verifiedAt,
	}).Error
	return &user, nil
}
