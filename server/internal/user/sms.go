package user

// 火山引擎短信发送（验证码）。
// 配置（.env）：VOLC_SMS_ACCESS_KEY / VOLC_SMS_SECRET_KEY / VOLC_SMS_ACCOUNT /
//               VOLC_SMS_SIGN / VOLC_SMS_TEMPLATE_ID
// 模板示例（变量名必须是 code）：您的验证码为${code}，5分钟内有效，请勿泄露。
// 签名按火山 V4 规范手写（官方 signature demo 移植），不引 SDK。

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type smsConfig struct {
	AccessKey  string
	SecretKey  string
	SmsAccount string
	Sign       string
	TemplateID string
}

func loadSmsConfig() smsConfig {
	return smsConfig{
		AccessKey:  os.Getenv("VOLC_SMS_ACCESS_KEY"),
		SecretKey:  os.Getenv("VOLC_SMS_SECRET_KEY"),
		SmsAccount: os.Getenv("VOLC_SMS_ACCOUNT"),
		Sign:       os.Getenv("VOLC_SMS_SIGN"),
		TemplateID: os.Getenv("VOLC_SMS_TEMPLATE_ID"),
	}
}

func (c smsConfig) configured() bool {
	return c.AccessKey != "" && c.SecretKey != "" && c.SmsAccount != "" && c.Sign != "" && c.TemplateID != ""
}

func hmacSHA256(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func hashSHA256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// sendSmsCode 发短信验证码；返回 error 时调用方应释放限频名额让用户可重试
func sendSmsCode(phone, code string) error {
	cfg := loadSmsConfig()
	body, _ := json.Marshal(map[string]string{
		"SmsAccount":    cfg.SmsAccount,
		"Sign":          cfg.Sign,
		"TemplateID":    cfg.TemplateID,
		"TemplateParam": fmt.Sprintf(`{"code":%q}`, code),
		"PhoneNumbers":  phone,
	})
	queries := url.Values{}
	queries.Set("Action", "SendSms")
	queries.Set("Version", "2020-01-01")
	req, err := http.NewRequest(http.MethodPost, "https://sms.volcengineapi.com/?"+queries.Encode(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	const (
		volcService = "volcSMS"
		volcRegion  = "cn-north-1"
	)
	date := time.Now().UTC().Format("20060102T150405Z")
	payload := hashSHA256Hex(body)
	req.Header.Set("X-Date", date)
	req.Header.Set("X-Content-Sha256", payload)
	req.Header.Set("Content-Type", "application/json")

	queryString := strings.Replace(queries.Encode(), "+", "%20", -1)
	signedHeaders := []string{"host", "x-date", "x-content-sha256", "content-type"}
	var hl []string
	for _, h := range signedHeaders {
		if h == "host" {
			hl = append(hl, h+":"+req.Host)
		} else {
			hl = append(hl, h+":"+strings.TrimSpace(req.Header.Get(h)))
		}
	}
	canonical := strings.Join([]string{
		http.MethodPost, "/", queryString,
		strings.Join(hl, "\n") + "\n",
		strings.Join(signedHeaders, ";"),
		payload,
	}, "\n")
	credentialScope := date[:8] + "/" + volcRegion + "/" + volcService + "/request"
	signString := strings.Join([]string{
		"HMAC-SHA256", date, credentialScope, hashSHA256Hex([]byte(canonical)),
	}, "\n")

	kSigning := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte(cfg.SecretKey), date[:8]), volcRegion), volcService), "request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, signString))
	req.Header.Set("Authorization", "HMAC-SHA256 Credential="+cfg.AccessKey+"/"+credentialScope+
		", SignedHeaders="+strings.Join(signedHeaders, ";")+", Signature="+signature)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		ResponseMetadata struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"ResponseMetadata"`
		Result struct {
			SendResults []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendResults"`
		} `json:"Result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if e := out.ResponseMetadata.Error; e != nil {
		return fmt.Errorf("volc sms: %s %s", e.Code, e.Message)
	}
	for _, r := range out.Result.SendResults {
		if r.Code != "" && r.Code != "OK" {
			return fmt.Errorf("volc sms: %s %s", r.Code, r.Message)
		}
	}
	return nil
}
