package user

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type mailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	Name     string
}

func loadMailConfig() mailConfig {
	from := os.Getenv("SMTP_FROM")
	username := os.Getenv("SMTP_USERNAME")
	if from == "" {
		from = username
	}
	name := os.Getenv("SMTP_FROM_NAME")
	if name == "" {
		name = "写作菌"
	}
	return mailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     getenvDefault("SMTP_PORT", "587"),
		Username: username,
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     from,
		Name:     name,
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateEmailCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashVerificationCode(targetType, target, scene, code string) string {
	salt := os.Getenv("VERIFICATION_CODE_SALT")
	if salt == "" {
		salt = os.Getenv("JWT_SECRET")
	}
	sum := sha256.Sum256([]byte(targetType + ":" + strings.ToLower(strings.TrimSpace(target)) + ":" + scene + ":" + code + ":" + salt))
	return fmt.Sprintf("%x", sum[:])
}

func verifyVerificationCodeHash(targetType, target, scene, code, expected string) bool {
	actual := hashVerificationCode(targetType, target, scene, code)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

// sceneText 场景文案：purpose(旧纯文本用) / title / desc
func sceneText(scene string) (purpose, title, desc string) {
	switch scene {
	case "register":
		return "注册", "注册验证码", "你正在注册写作菌账号"
	case "reset_password":
		return "重置密码", "重置密码验证码", "你正在重置写作菌账号密码"
	default:
		return "登录", "登录验证码", "你正在登录写作菌"
	}
}

// buildCodeEmailHTML 方案 B「大字极简」邮件模板（全内联样式，兼容各邮件客户端）
func buildCodeEmailHTML(title, desc, code string) string {
	return `<div style="max-width:420px;margin:0 auto;background:#ffffff;padding:40px 30px;font-family:-apple-system,'PingFang SC','Microsoft YaHei',sans-serif">` +
		`<div style="font-size:13px;letter-spacing:3px;color:#999999">XIEZUOJUN</div>` +
		`<div style="font-size:22px;font-weight:700;color:#111111;margin:16px 0 4px">` + title + `</div>` +
		`<div style="font-size:13px;color:#888888">` + desc + `</div>` +
		`<div style="font-size:52px;font-weight:800;letter-spacing:14px;color:#111111;margin:26px 0">` + code + `</div>` +
		`<div style="border-top:1px solid #eeeeee;margin:22px 0"></div>` +
		`<div style="font-size:12px;color:#aaaaaa;line-height:1.8">10 分钟内有效 · 请勿泄露<br>若非本人操作，请忽略这封邮件</div>` +
		`</div>`
}

// buildCodeEmailMsg 组装 multipart/alternative 邮件（HTML + 纯文本兜底）
func buildCodeEmailMsg(fromHeader, to, subject, plainBody, htmlBody string) []byte {
	const boundary = "xzj-mail-boundary-7f3a9c"
	return []byte(strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"",
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		plainBody,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"",
		htmlBody,
		"--" + boundary + "--",
	}, "\r\n"))
}

func sendEmailCode(to, scene, code string) error {
	cfg := loadMailConfig()
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" || cfg.From == "" {
		log.Printf("[email-code] SMTP not configured scene=%s to=%s code=%s", scene, to, code)
		return nil
	}

	purpose, title, desc := sceneText(scene)
	subject := "写作菌" + purpose + "验证码"
	plainBody := fmt.Sprintf("您的写作菌%s验证码是：%s\n\n验证码 10 分钟内有效。若非本人操作，请忽略这封邮件。", purpose, code)
	htmlBody := buildCodeEmailHTML(title, desc, code)
	fromHeader := cfg.From
	if cfg.Name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", cfg.Name, cfg.From)
	}
	msg := buildCodeEmailMsg(fromHeader, to, subject, plainBody, htmlBody)

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port <= 0 {
		port = 587
	}
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	addr := fmt.Sprintf("%s:%d", cfg.Host, port)
	if port == 465 {
		return sendMailTLS(addr, cfg.Host, auth, cfg.From, []string{to}, []byte(msg))
	}
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
