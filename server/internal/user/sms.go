package user

// 腾讯云短信发送（验证码）。
// 配置（.env）：TENCENT_SMS_SECRET_ID / TENCENT_SMS_SECRET_KEY / TENCENT_SMS_SDK_APP_ID /
//               TENCENT_SMS_SIGN_NAME / TENCENT_SMS_TEMPLATE_ID
// 模板需含 {1} 占位验证码，如：您的验证码为{1}，10分钟内有效，请勿泄露。

import (
	"fmt"
	"os"
	"sync"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type smsConfig struct {
	SecretID   string
	SecretKey  string
	SdkAppID   string
	SignName   string
	TemplateID string
}

func loadSmsConfig() smsConfig {
	return smsConfig{
		SecretID:   os.Getenv("TENCENT_SMS_SECRET_ID"),
		SecretKey:  os.Getenv("TENCENT_SMS_SECRET_KEY"),
		SdkAppID:   os.Getenv("TENCENT_SMS_SDK_APP_ID"),
		SignName:   os.Getenv("TENCENT_SMS_SIGN_NAME"),
		TemplateID: os.Getenv("TENCENT_SMS_TEMPLATE_ID"),
	}
}

func (c smsConfig) configured() bool {
	return c.SecretID != "" && c.SecretKey != "" && c.SdkAppID != "" && c.SignName != "" && c.TemplateID != ""
}

var (
	smsClientOnce sync.Once
	smsClient     *sms.Client
	smsClientErr  error
)

func getSmsClient(cfg smsConfig) (*sms.Client, error) {
	smsClientOnce.Do(func() {
		cred := common.NewCredential(cfg.SecretID, cfg.SecretKey)
		cp := profile.NewClientProfile()
		cp.HttpProfile.Endpoint = "sms.tencentcloudapi.com"
		smsClient, smsClientErr = sms.NewClient(cred, "ap-guangzhou", cp)
	})
	return smsClient, smsClientErr
}

// sendSmsCode 发短信验证码；返回 error 时调用方应释放限频名额让用户可重试
func sendSmsCode(phone, code string) error {
	cfg := loadSmsConfig()
	client, err := getSmsClient(cfg)
	if err != nil {
		return err
	}
	req := sms.NewSendSmsRequest()
	req.SmsSdkAppId = common.StringPtr(cfg.SdkAppID)
	req.SignName = common.StringPtr(cfg.SignName)
	req.TemplateId = common.StringPtr(cfg.TemplateID)
	req.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})
	req.TemplateParamSet = common.StringPtrs([]string{code})
	resp, err := client.SendSms(req)
	if err != nil {
		return err
	}
	for _, st := range resp.Response.SendStatusSet {
		if st.Code != nil && *st.Code != "Ok" {
			msg := ""
			if st.Message != nil {
				msg = *st.Message
			}
			return fmt.Errorf("sms send failed: %s %s", *st.Code, msg)
		}
	}
	return nil
}
