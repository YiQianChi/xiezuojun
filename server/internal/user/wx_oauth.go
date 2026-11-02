package user

import (
	"encoding/json"
	"errors"
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

// 微信扫码登录（开放平台网站应用，PC 网页弹二维码）。
// 与小程序 wx.login 是两条链路，但 unionid 相同即同一账号（小程序需绑定到同一开放平台）。
// 配置：WX_OAUTH_APPID / WX_OAUTH_SECRET（开放平台网站应用，审核通过后填入 .env 即点亮）。
// 回调地址：https://xiezuojun.com/api/auth/wx-oauth/callback（回调域 xiezuojun.com 要在开放平台登记）

func (h *Handler) oauthConfigured() bool {
	return h.wxOauthAppID != "" && h.wxOauthSecret != ""
}

// WxOauthStart 发起微信扫码登录：跳微信官方二维码页，用户扫码确认后带 code 回调
func (h *Handler) WxOauthStart(c *gin.Context) {
	if !h.oauthConfigured() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "wechat oauth not configured"})
		return
	}
	redirect := c.Query("redirect")
	if redirect == "" || !strings.HasPrefix(redirect, "http") {
		redirect = "https://xiezuojun.com/login"
	}
	// embed=1：二维码嵌在登录页 iframe 里，回调时 postMessage 回父页面而不是整页跳转
	state := redirect
	if c.Query("embed") == "1" {
		state += "|e"
	}
	// bind_ticket：登录态下扫码绑定微信（个人设置发起），回调走绑定分支
	if t := c.Query("bind_ticket"); t != "" {
		state += "|b:" + t
	}
	cb := "https://xiezuojun.com/api/auth/wx-oauth/callback"
	u := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		h.wxOauthAppID, url.QueryEscape(cb), url.QueryEscape(state),
	)
	c.Redirect(http.StatusFound, u)
}

type oauthTokenResp struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WxOauthCallback 微信授权回调：code 换 openid/unionid，建号/登录，token 桥回前端
func (h *Handler) WxOauthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	rawState := c.Query("state")
	bindTicket := ""
	embed := false
	parts := strings.Split(rawState, "|")
	front := parts[0]
	for _, p := range parts[1:] {
		if p == "e" {
			embed = true
		}
		if strings.HasPrefix(p, "b:") {
			bindTicket = strings.TrimPrefix(p, "b:")
		}
	}
	if !strings.HasPrefix(front, "http") {
		front = "https://xiezuojun.com/login"
	}
	// ---- 绑定分支：个人设置里扫码绑微信，结果 postMessage 回父页面 ----
	if bindTicket != "" {
		bindFail := func(msg string) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(
				`<!doctype html><meta charset="utf-8"><script>parent.postMessage({type:'xzj-wx-bind',error:%q},location.origin)</script><p style="font:14px sans-serif;text-align:center;margin-top:40px">%s</p>`, msg, msg)))
		}
		uid, err := vcodeRdb.Get(vcodeCtx, "xzj:wxbind:"+bindTicket).Result()
		if err != nil {
			bindFail("绑定会话已过期，请重新发起绑定")
			return
		}
		vcodeRdb.Del(vcodeCtx, "xzj:wxbind:"+bindTicket)
		if !h.oauthConfigured() || code == "" {
			bindFail("微信配置异常")
			return
		}
		tok, err := h.oauthGetToken(code)
		if err != nil {
			bindFail("微信授权失败")
			return
		}
		identifier := tok.UnionID
		if identifier == "" {
			identifier = tok.OpenID
		}
		if identifier == "" {
			bindFail("微信返回身份为空")
			return
		}
		if err := h.bindWechatToUser(uid, identifier); err != nil {
			if errors.Is(err, errAccountConflict) {
				bindFail("该微信已绑定其他账号")
				return
			}
			bindFail("绑定失败：" + err.Error())
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
			`<!doctype html><meta charset="utf-8"><script>parent.postMessage({type:'xzj-wx-bind',ok:true},location.origin)</script><p style="font:14px sans-serif;text-align:center;margin-top:40px">绑定成功</p>`))
		return
	}

	// ---- 登录分支 ----
	// embed 模式：成功/失败都回传结果。微信扫码页确认后是顶层跳转（frame-busting），
	// 此时 iframe 已不存在，检测到自己不在 iframe 里就直接存 token 跳前端
	if embed {
		wrap := func(jsPayload, hint string) string {
			return fmt.Sprintf(`<!doctype html><meta charset="utf-8"><script>
var p=%s;
if (window !== window.top) {
  try { parent.postMessage(p, location.origin) } catch(e) {}
} else if (p.at) {
  localStorage.setItem('access_token', p.at);
  if (p.rt) localStorage.setItem('refresh_token', p.rt);
  location.href = '/bookshelf';
} else {
  location.href = %q + (p.error ? ('?wx_err=' + encodeURIComponent(p.error)) : '');
}
</script><p style="font:14px sans-serif;text-align:center;margin-top:40px">%s</p>`, jsPayload, front, hint)
		}
		fail := func(msg string) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wrap(
				fmt.Sprintf(`{type:'xzj-wx-login',error:%q}`, msg), msg)))
		}
		if !h.oauthConfigured() || code == "" {
			fail("微信登录配置异常")
			return
		}
		tok, err := h.oauthGetToken(code)
		if err != nil {
			fail("微信授权失败：" + err.Error())
			return
		}
		user, err := h.wechatUserFromOAuthToken(tok)
		if err != nil {
			fail(err.Error())
			return
		}
		access, refresh, err := h.issueTokens(c, user)
		if err != nil {
			fail("生成登录态失败")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(wrap(
			fmt.Sprintf(`{type:'xzj-wx-login',at:%q,rt:%q}`, access, refresh), "登录成功，正在进入…")))
		return
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
	user, err := h.wechatUserFromOAuthToken(tok)
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

// wechatUserFromOAuthToken 用 oauth token 的 unionid/openid 查号/建号（回调两种模式共用）
func (h *Handler) wechatUserFromOAuthToken(tok *oauthTokenResp) (*model.User, error) {
	if tok.UnionID == "" && tok.OpenID == "" {
		return nil, fmt.Errorf("empty openid")
	}
	if user, err := h.findWechatUser(tok.UnionID, tok.OpenID); err != nil {
		return nil, err
	} else if user != nil {
		return user, nil
	}
	identifier := tok.UnionID
	if identifier == "" {
		identifier = tok.OpenID
	}
	return h.findOrCreateWechatUser(identifier)
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
