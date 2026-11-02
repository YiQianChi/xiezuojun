package user

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	goclick "github.com/wenlng/go-captcha/v2/click"
)

func timeNowDay() string { return time.Now().Format("20060102") }

// GoCaptcha 点选：阈值触发、按序命中、官方校验、时序、一次性消费
func TestCaptcha(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	old := vcodeRdb
	vcodeRdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { vcodeRdb = old; mr.Close() }()

	// 阈值触发（号/IP 两维度）
	if smsNeedsCaptcha("13800000001", "1.1.1.1") {
		t.Fatal("零发送不应要求验证码")
	}
	d := timeNowDay()
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("phone", "13800000001", d), "2", 0)
	if !smsNeedsCaptcha("13800000001", "9.9.9.9") {
		t.Fatal("同号第 3 条应要求验证码")
	}
	vcodeRdb.Set(vcodeCtx, smsQuotaKey("ip", "1.1.1.1", d), "2", 0)
	if !smsNeedsCaptcha("13800000002", "1.1.1.1") {
		t.Fatal("同 IP 第 3 个号应要求验证码")
	}

	// 出题：图片+提示条生成，答案点阵入 Redis
	ch := newClickCaptcha()
	if ch.Img == "" || ch.Thumb == "" || ch.N < 2 {
		t.Fatalf("应生成图片与提示条: n=%d", ch.N)
	}
	var dots map[int]*goclick.Dot
	raw, _ := mr.Get(captchaPrefix + ch.ID)
	if err := json.Unmarshal([]byte(raw), &dots); err != nil || len(dots) != ch.N {
		t.Fatalf("答案点阵应存 Redis: %v", err)
	}

	// 按序点中（用点阵中心坐标）→ 过
	pts := make([][2]int, 0, len(dots))
	for i := 0; i < len(dots); i++ {
		pts = append(pts, [2]int{dots[i].X, dots[i].Y})
	}
	ok, _ := json.Marshal(clickAnswer{Points: pts, Dur: 1500})
	if !verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("按顺序点中应通过")
	}
	if verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("验证码应一次性消费")
	}

	// 顺序颠倒 → 拒
	ch2 := newClickCaptcha()
	var d2 map[int]*goclick.Dot
	raw2, _ := mr.Get(captchaPrefix + ch2.ID)
	json.Unmarshal([]byte(raw2), &d2)
	n := len(d2)
	swapped := make([][2]int, 0, n)
	for i := n - 1; i >= 0; i-- {
		swapped = append(swapped, [2]int{d2[i].X, d2[i].Y})
	}
	bad, _ := json.Marshal(clickAnswer{Points: swapped, Dur: 1500})
	if n > 1 && verifyCaptcha(ch2.ID, string(bad)) {
		t.Fatal("顺序点错应拒绝")
	}

	// 100ms 瞬时点完（机器）→ 拒
	ch3 := newClickCaptcha()
	var d3 map[int]*goclick.Dot
	raw3, _ := mr.Get(captchaPrefix + ch3.ID)
	json.Unmarshal([]byte(raw3), &d3)
	fastPts := make([][2]int, 0, len(d3))
	for i := 0; i < len(d3); i++ {
		fastPts = append(fastPts, [2]int{d3[i].X, d3[i].Y})
	}
	fast, _ := json.Marshal(clickAnswer{Points: fastPts, Dur: 100})
	if verifyCaptcha(ch3.ID, string(fast)) {
		t.Fatal("100ms 点完应拒绝")
	}

	if verifyCaptcha("nonexistent", "{}") {
		t.Fatal("不存在的 id 不应通过")
	}
}
