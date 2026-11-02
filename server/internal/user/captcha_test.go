package user

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func timeNowDay() string { return time.Now().Format("20060102") }

// solveCaptchaForTest 解析 "a + b = ?" / "a - b = ?"
func solveCaptchaForTest(q string) string {
	parts := strings.Fields(strings.TrimSuffix(q, "= ?"))
	a, _ := strconv.Atoi(parts[0])
	b, _ := strconv.Atoi(parts[2])
	if parts[1] == "+" {
		return strconv.Itoa(a + b)
	}
	return strconv.Itoa(a - b)
}

// 渐进式验证码：阈值触发（号/IP 两个维度）、题目可验证、一次性消费
func TestCaptcha(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	old := vcodeRdb
	vcodeRdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { vcodeRdb = old; mr.Close() }()

	if smsNeedsCaptcha("13800000001", "1.1.1.1") {
		t.Fatal("零发送不应要求验证码")
	}
	d := timeNowDay()
	// 号维度：当日该号已发 2 条 → 触发（换 IP 也拦）
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("phone", "13800000001", d), "2", 0)
	if !smsNeedsCaptcha("13800000001", "9.9.9.9") {
		t.Fatal("同号第 3 条应要求验证码（即使换 IP）")
	}
	// IP 维度：当日同 IP 已发 2 条 → 对新号也触发
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("ip", "1.1.1.1", d), "2", 0)
	if !smsNeedsCaptcha("13800000002", "1.1.1.1") {
		t.Fatal("同 IP 第 3 个号应要求验证码")
	}
	if smsNeedsCaptcha("13800000002", "2.2.2.2") {
		t.Fatal("新号+新 IP 不应要求验证码")
	}

	// 题目：答对一次通过，一次性消费
	id, q := newCaptcha()
	if q == "" || id == "" {
		t.Fatal("应生成题目")
	}
	ans := solveCaptchaForTest(q)
	if !verifyCaptcha(id, ans) {
		t.Fatalf("正确答案应通过（题=%s 答=%s）", q, ans)
	}
	if verifyCaptcha(id, ans) {
		t.Fatal("验证码应一次性消费")
	}
	if verifyCaptcha("nonexistent", "1") {
		t.Fatal("不存在的 id 不应通过")
	}
}
