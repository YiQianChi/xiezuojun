// 验证码 Redis 存储 — 替代 verification_codes 表
// 键设计：
//
//	xzj:vcode:{scene}:{target}        验证码哈希，TTL 10 分钟（过期自动失效，不再堆垃圾行）
//	xzj:vcode:limit:{scene}:{target}  发送限频，SET NX EX 60（60 秒内只能发一次）
package user

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const (
	vcodeKeyPrefix  = "xzj:vcode:"
	vcodeRatePrefix = "xzj:vcode:limit:"
	vcodeTryPrefix  = "xzj:vcode:try:"
	vcodeTTL        = 10 * time.Minute
	vcodeRateTTL    = 60 * time.Second
	maxCodeTries    = 5
)

var (
	vcodeRdb *redis.Client
	vcodeCtx = context.Background()
)

func init() {
	// ponytail-desktop: DESKTOP=1 时用内存 miniredis（发码本身在桌面版用不到，仅保启动不炸）
	if os.Getenv("DESKTOP") == "1" {
		mr := miniredis.NewMiniRedis()
		if err := mr.Start(); err != nil {
			panic("miniredis start failed: " + err.Error())
		}
		vcodeRdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		return
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = n
		}
	}
	vcodeRdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
}

// ---- 短信防刷配额（防黑产刷量/短信轰炸）：单号、单IP、全局三档，窗口固定时间片 ----

// smsDailyCap 全站每日短信熔断线，超了直接拒绝（保护预算），可用 SMS_DAILY_CAP 覆盖
var smsDailyCap = func() int64 {
	if v := os.Getenv("SMS_DAILY_CAP"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 2000
}()

func smsQuotaKey(dim, id, win string) string {
	return "xzj:sms:q:" + dim + ":" + id + ":" + win
}

// acquireSmsQuota 先占后发：占用单号/单IP/全局各窗口配额，返回超限说明（空串=放行）。
// 超限时回退本次已占的名额，保证各窗口计数一致。
func acquireSmsQuota(phone, ip string) string {
	now := time.Now()
	h, d := now.Format("2006010215"), now.Format("20060102")
	quotas := []struct {
		key    string
		limit  int64
		hourly bool
	}{
		{smsQuotaKey("phone", phone, h), 5, true},
		{smsQuotaKey("phone", phone, d), 10, false},
		{smsQuotaKey("ip", ip, h), 10, true},
		{smsQuotaKey("ip", ip, d), 30, false},
		{smsQuotaKey("global", "", d), smsDailyCap, false},
	}
	var taken []string
	for _, q := range quotas {
		n, err := vcodeRdb.Incr(vcodeCtx, q.key).Result()
		if err != nil {
			continue // redis 异常不挡正常用户（会丢防线，但可用性优先）
		}
		taken = append(taken, q.key)
		if n == 1 {
			if q.hourly {
				vcodeRdb.Expire(vcodeCtx, q.key, 2*time.Hour)
			} else {
				vcodeRdb.Expire(vcodeCtx, q.key, 48*time.Hour)
			}
		}
		if n > q.limit {
			for _, k := range taken {
				vcodeRdb.Decr(vcodeCtx, k)
			}
			return "limit"
		}
	}
	return ""
}

// releaseSmsQuota 发送失败时回退配额，别让失败的发送烧名额
func releaseSmsQuota(phone, ip string) {
	now := time.Now()
	h, d := now.Format("2006010215"), now.Format("20060102")
	for _, k := range []string{
		smsQuotaKey("phone", phone, h), smsQuotaKey("phone", phone, d),
		smsQuotaKey("ip", ip, h), smsQuotaKey("ip", ip, d),
		smsQuotaKey("global", "", d),
	} {
		vcodeRdb.Decr(vcodeCtx, k)
	}
}

func vcodeKey(scene, target string) string     { return vcodeKeyPrefix + scene + ":" + target }
func vcodeRateKey(scene, target string) string { return vcodeRatePrefix + scene + ":" + target }

// acquireSendSlot 尝试占发送名额：true=允许发送（60s 内首次），false=太频繁
func acquireSendSlot(scene, target string) (bool, error) {
	return vcodeRdb.SetNX(vcodeCtx, vcodeRateKey(scene, target), 1, vcodeRateTTL).Result()
}

// storeCode 保存验证码哈希（覆盖旧码，等价于旧逻辑"取最新一行"）
func storeCode(scene, target, codeHash string) error {
	return vcodeRdb.Set(vcodeCtx, vcodeKey(scene, target), codeHash, vcodeTTL).Err()
}

// releaseSendSlot 发送失败时释放限频名额和已存验证码，让用户能立即重试
func releaseSendSlot(scene, target string) {
	vcodeRdb.Del(vcodeCtx, vcodeRateKey(scene, target), vcodeKey(scene, target))
}

// getCode 取出验证码哈希（不销毁，输错可重试）；键不存在=过期/未发送
func getCode(scene, target string) (string, bool) {
	hash, err := vcodeRdb.Get(vcodeCtx, vcodeKey(scene, target)).Result()
	if err != nil {
		return "", false
	}
	return hash, true
}

// burnCode 校验成功后销毁验证码（一次性使用）
func burnCode(scene, target string) {
	vcodeRdb.Del(vcodeCtx, vcodeKey(scene, target), vcodeTryPrefix+scene+":"+target)
}

// incrTry 失败尝试计数，超过 maxCodeTries 次后验证码作废（防 6 位码爆破）
func incrTry(scene, target string) (int64, error) {
	key := vcodeTryPrefix + scene + ":" + target
	n, err := vcodeRdb.Incr(vcodeCtx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		vcodeRdb.Expire(vcodeCtx, key, vcodeTTL)
	}
	return n, nil
}
