package user

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func timeNowWindow() string { return time.Now().Format("2006010215") }

// 防刷配额：单号 5/时、单IP 10/时、全站熔断、超限回退计数
func TestSmsQuota(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	old := vcodeRdb
	vcodeRdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { vcodeRdb = old; mr.Close() }()

	// 同一手机号前 5 次放行，第 6 次拒绝
	for i := 1; i <= 5; i++ {
		if r := acquireSmsQuota("13800000001", "1.1.1.1"); r != "" {
			t.Fatalf("第 %d 次应放行", i)
		}
	}
	if r := acquireSmsQuota("13800000001", "1.1.1.1"); r == "" {
		t.Fatal("第 6 次应被单号小时配额拒绝")
	}

	// 换新号仍受同一 IP 小时配额约束：IP 已占 5，再放 5 个号到 10，第 6 个号第 11 次被拒
	for i := 1; i <= 5; i++ {
		if r := acquireSmsQuota("1380000000"+string(rune('0'+i+1)), "1.1.1.1"); r != "" {
			t.Fatalf("新号第 %d 次应放行", i)
		}
	}
	if r := acquireSmsQuota("13800000009", "1.1.1.1"); r == "" {
		t.Fatal("IP 小时配额（10）用尽后应拒绝")
	}

	// 发送失败回退：占一次再退一次，当前小时窗计数应为 0
	h := timeNowWindow()
	acquireSmsQuota("13900000000", "2.2.2.2")
	releaseSmsQuota("13900000000", "2.2.2.2")
	if n, _ := vcodeRdb.Get(vcodeCtx, smsQuotaKey("phone", "13900000000", h)).Result(); n != "0" && n != "" {
		t.Fatalf("回退后计数应为 0 或不存在，得到 %s", n)
	}
}
