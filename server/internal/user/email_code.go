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

func sendEmailCode(to, scene, code string) error {
	cfg := loadMailConfig()
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" || cfg.From == "" {
		log.Printf("[email-code] SMTP not configured scene=%s to=%s code=%s", scene, to, code)
		return nil
	}

	subject := "写作菌验证码"
	purpose := "登录"
	if scene == "register" {
		purpose = "注册"
	} else if scene == "reset_password" {
		purpose = "重置密码"
	}
	body := fmt.Sprintf("您的写作菌%s验证码是：%s\n\n验证码 10 分钟内有效。若非本人操作，请忽略这封邮件。", purpose, code)
	fromHeader := cfg.From
	if cfg.Name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", cfg.Name, cfg.From)
	}
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

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
