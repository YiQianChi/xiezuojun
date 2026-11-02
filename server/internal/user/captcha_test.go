package user

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func timeNowDay() string { return time.Now().Format("20060102") }

// 点选验证码：阈值触发（号/IP 两维度）、顺序命中、容差、时长、一次性消费
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

	// 从 Redis 读答案坐标，按顺序点击 → 过
	ch := newClickCaptcha()
	var target [][2]int
	rawAns, _ := mr.Get(captchaPrefix + ch.ID)
	if err := json.Unmarshal([]byte(rawAns), &target); err != nil || len(target) != 3 {
		t.Fatalf("答案应存 3 个坐标: %v", err)
	}
	ok, _ := json.Marshal(clickAnswer{Points: target, Dur: 1200})
	if !verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("按顺序点中应通过")
	}
	if verifyCaptcha(ch.ID, string(ok)) {
		t.Fatal("验证码应一次性消费")
	}

	// 顺序点错（交换前两个目标，目标间距>70px 超容差 24px）→ 拒
	ch2 := newClickCaptcha()
	var t2 [][2]int
	raw2, _ := mr.Get(captchaPrefix + ch2.ID)
	json.Unmarshal([]byte(raw2), &t2)
	bad, _ := json.Marshal(clickAnswer{Points: [][2]int{t2[1], t2[0], t2[2]}, Dur: 1200})
	if verifyCaptcha(ch2.ID, string(bad)) {
		t.Fatal("顺序点错应拒绝")
	}

	// 100ms 瞬时点完（机器）→ 拒
	ch3 := newClickCaptcha()
	var t3 [][2]int
	raw3, _ := mr.Get(captchaPrefix + ch3.ID)
	json.Unmarshal([]byte(raw3), &t3)
	fast, _ := json.Marshal(clickAnswer{Points: t3, Dur: 100})
	if verifyCaptcha(ch3.ID, string(fast)) {
		t.Fatal("100ms 点完应拒绝")
	}

	if verifyCaptcha("nonexistent", "{}") {
		t.Fatal("不存在的 id 不应通过")
	}
}
