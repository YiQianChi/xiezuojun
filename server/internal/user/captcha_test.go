package user

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func timeNowDay() string { return time.Now().Format("20060102") }

// humanTrail 模拟人手拖动：多采样、总耗时 >=200ms
func humanTrail(from, to, ms int) [][2]int {
	trail := [][2]int{}
	for i := 0; i <= 8; i++ {
		trail = append(trail, [2]int{ms * i / 8, from + (to-from)*i/8})
	}
	return trail
}

// 滑块验证码：阈值触发（号/IP 两维度）、落点容差、轨迹校验、一次性消费
func TestCaptcha(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	old := vcodeRdb
	vcodeRdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { vcodeRdb = old; mr.Close() }()

	// 阈值触发
	if smsNeedsCaptcha("13800000001", "1.1.1.1") {
		t.Fatal("零发送不应要求验证码")
	}
	d := timeNowDay()
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("phone", "13800000001", d), "2", 0)
	if !smsNeedsCaptcha("13800000001", "9.9.9.9") {
		t.Fatal("同号第 3 条应要求验证码（即使换 IP）")
	}
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("ip", "1.1.1.1", d), "2", 0)
	if !smsNeedsCaptcha("13800000002", "1.1.1.1") {
		t.Fatal("同 IP 第 3 个号应要求验证码")
	}
	if smsNeedsCaptcha("13800000002", "2.2.2.2") {
		t.Fatal("新号+新 IP 不应要求验证码")
	}

	// 从 Redis 读答案构造合法提交：落点准 + 人类轨迹 → 过
	ch := newSliderCaptcha()
	want, _ := mr.Get(captchaPrefix + ch.ID)
	target, _ := strconv.Atoi(want)
	ok, _ := json.Marshal(captchaAnswer{X: target, Trail: humanTrail(24, target, 900)})
	if !verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("落点准确+人类轨迹应通过")
	}
	if verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("验证码应一次性消费")
	}

	// 落点超差 → 拒
	ch2 := newSliderCaptcha()
	w2, _ := mr.Get(captchaPrefix + ch2.ID)
	t2, _ := strconv.Atoi(w2)
	bad, _ := json.Marshal(captchaAnswer{X: t2 + 30, Trail: humanTrail(24, t2+30, 900)})
	if verifyCaptcha(ch2.ID, string(bad)) {
		t.Fatal("落点偏差 30px 应拒绝")
	}

	// 轨迹采样少+瞬时完成（机器直填）→ 拒
	ch3 := newSliderCaptcha()
	w3, _ := mr.Get(captchaPrefix + ch3.ID)
	t3, _ := strconv.Atoi(w3)
	fast, _ := json.Marshal(captchaAnswer{X: t3, Trail: [][2]int{{0, 24}, {10, t3}}})
	if verifyCaptcha(ch3.ID, string(fast)) {
		t.Fatal("2 个采样+10ms 的机器轨迹应拒绝")
	}

	if verifyCaptcha("nonexistent", "{}") {
		t.Fatal("不存在的 id 不应通过")
	}
}
