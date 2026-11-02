package user

// 微信小程序手机号一键登录/绑定（企业主体，getPhoneNumber）。
// 前端同时提交 wx.login 的 code 和 button open-type="getPhoneNumber" 回调的 code。
// 后端用微信返回的手机号匹配已有账号：命中即把微信身份并到老账号；
// 微信自动建的孤儿号（除微信外无其他登录方式）会被合并——作品迁到老账号、孤儿号禁用。

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

type WxPhoneLoginReq struct {
	Code      string `json:"code" binding:"required"`       // wx.login
	PhoneCode string `json:"phone_code" binding:"required"` // getPhoneNumber 回调 e.detail.code
}

var errAccountConflict = errors.New("wechat bound to another full account")

// WxPhoneLogin 小程序手机号一键登录/绑定
func (h *Handler) WxPhoneLogin(c *gin.Context) {
	if !h.wxConfigured() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "wechat login not configured"})
		return
	}
	var req WxPhoneLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and phone_code required"})
		return
	}

	sess, err := h.jscode2session(strings.TrimSpace(req.Code))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wechat login failed: " + err.Error()})
		return
	}
	identifier := sess.OpenID
	if sess.UnionID != "" {
		identifier = sess.UnionID
	}
	if identifier == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wechat login failed: empty openid"})
		return
	}

	phone, err := h.wxGetPhoneNumber(strings.TrimSpace(req.PhoneCode))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "get phone failed: " + err.Error()})
		return
	}

	user, isNew, merged, err := h.bindWechatPhone(identifier, phone)
	if err != nil {
		switch {
		case errors.Is(err, errAccountConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "wechat already bound to another account"})
		case err.Error() == "account disabled":
			c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
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
			"merged":        merged, // true=微信孤儿号已并入手机号老账号
		},
	})
}

// bindWechatPhone 微信身份 + 手机号落同一账号。返回登录用户、是否新注册、是否发生孤儿号合并。
func (h *Handler) bindWechatPhone(wxIdentifier, phone string) (user *model.User, isNew, merged bool, err error) {
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 微信身份现状
		var wxIdentity model.AuthIdentity
		hasWx := tx.Where("provider = ? AND identifier = ?", "wechat", wxIdentifier).First(&wxIdentity).Error == nil

		// 手机号对应的已有账号（identity 优先，users.phone 兜底）
		var phoneUser model.User
		hasPhoneUser := false
		var phoneIdentity model.AuthIdentity
		if e := tx.Where("provider = ? AND identifier = ?", "phone", phone).First(&phoneIdentity).Error; e == nil {
			hasPhoneUser = tx.Where("user_id = ? AND status = 1", phoneIdentity.UserID).First(&phoneUser).Error == nil
		} else {
			hasPhoneUser = tx.Where("phone = ? AND status = 1", phone).First(&phoneUser).Error == nil
		}

		target := model.User{}
		switch {
		case hasPhoneUser:
			target = phoneUser
			if hasWx && wxIdentity.UserID != phoneUser.UserID {
				// 孤儿号合并：仅当孤儿号除微信外没有其他登录方式，否则不能静默并号
				var other int64
				tx.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider <> 'wechat'", wxIdentity.UserID).Count(&other)
				var orphan model.User
				if e := tx.Where("user_id = ?", wxIdentity.UserID).First(&orphan).Error; e != nil {
					return e
				}
				if other > 0 || orphan.IsAdmin {
					return errAccountConflict
				}
				if e := tx.Model(&model.Project{}).Where("user_id = ?", orphan.UserID).Update("user_id", phoneUser.UserID).Error; e != nil {
					return e
				}
				if e := tx.Model(&model.AuthIdentity{}).Where("id = ?", wxIdentity.ID).Update("user_id", phoneUser.UserID).Error; e != nil {
					return e
				}
				if e := tx.Model(&model.User{}).Where("user_id = ?", orphan.UserID).Update("status", 0).Error; e != nil {
					return e
				}
				merged = true
			} else if !hasWx {
				// 微信身份首次：直接挂到老账号
				verifiedAt := time.Now()
				if e := tx.Create(&model.AuthIdentity{
					UserID: phoneUser.UserID, Provider: "wechat", Identifier: wxIdentifier, VerifiedAt: &verifiedAt,
				}).Error; e != nil {
					return e
				}
			}
			if target.Phone == "" {
				tx.Model(&model.User{}).Where("user_id = ?", target.UserID).Update("phone", phone)
			}
			ensurePhoneIdentity(tx, target.UserID, phone)

		case hasWx:
			// 手机号没注册过、微信已建号：给现有微信账号补手机身份
			if e := tx.Where("user_id = ? AND status = 1", wxIdentity.UserID).First(&target).Error; e != nil {
				return errors.New("account disabled")
			}
			if e := tx.Model(&model.User{}).Where("user_id = ?", target.UserID).Update("phone", phone).Error; e != nil {
				return e
			}
			ensurePhoneIdentity(tx, target.UserID, phone)

		default:
			// 全新用户：建号 + 微信/手机双身份
			target = model.User{
				UserID:   uuid.NewString(),
				Phone:    phone,
				Nickname: randomPenName(),
				Plan:     "free",
				Status:   1,
			}
			if e := tx.Create(&target).Error; e != nil {
				return e
			}
			verifiedAt := time.Now()
			tx.Create(&model.AuthIdentity{UserID: target.UserID, Provider: "wechat", Identifier: wxIdentifier, VerifiedAt: &verifiedAt})
			tx.Create(&model.AuthIdentity{UserID: target.UserID, Provider: "phone", Identifier: phone, VerifiedAt: &verifiedAt})
			isNew = true
		}
		user = &target
		return nil
	})
	return
}

// ---- 微信服务端 access_token（7200s 有效，提前 5 分钟刷新）----

var (
	wxTokenMu     sync.Mutex
	wxToken       string
	wxTokenExpire time.Time
)

func (h *Handler) wxAccessToken() (string, error) {
	wxTokenMu.Lock()
	defer wxTokenMu.Unlock()
	if wxToken != "" && time.Now().Before(wxTokenExpire) {
		return wxToken, nil
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		h.wxAppID, h.wxAppSecret,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return "", fmt.Errorf("weixin token errcode=%d errmsg=%s", out.ErrCode, out.ErrMsg)
	}
	wxToken = out.AccessToken
	wxTokenExpire = time.Now().Add(time.Duration(out.ExpiresIn-300) * time.Second)
	return wxToken, nil
}

type wxPhoneInfoResp struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

func (h *Handler) wxGetPhoneNumber(phoneCode string) (string, error) {
	body, _ := json.Marshal(map[string]string{"code": phoneCode})
	call := func() (*wxPhoneInfoResp, error) {
		token, err := h.wxAccessToken()
		if err != nil {
			return nil, err
		}
		u := "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=" + token
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(u, "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var out wxPhoneInfoResp
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	out, err := call()
	if err != nil {
		return "", err
	}
	if out.ErrCode == 40001 || out.ErrCode == 42001 {
		// token 失效（多实例/被别处刷新）：清缓存重试一次
		wxTokenMu.Lock()
		wxToken = ""
		wxTokenMu.Unlock()
		if out, err = call(); err != nil {
			return "", err
		}
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("weixin getuserphonenumber errcode=%d errmsg=%s", out.ErrCode, out.ErrMsg)
	}
	phone := out.PhoneInfo.PurePhoneNumber
	if out.PhoneInfo.CountryCode != "" && out.PhoneInfo.CountryCode != "86" {
		phone = out.PhoneInfo.PhoneNumber // 非大陆号保留国家码，避免与国内号撞号
	}
	if phone == "" {
		return "", errors.New("empty phone number")
	}
	return phone, nil
}
