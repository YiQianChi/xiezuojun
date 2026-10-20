// 验证码 Redis 存储 — 替代 verification_codes 表
// 键设计：
//   xzj:vcode:{scene}:{target}        验证码哈希，TTL 10 分钟（过期自动失效，不再堆垃圾行）
//   xzj:vcode:limit:{scene}:{target}  发送限频，SET NX EX 60（60 秒内只能发一次）
package user

import (
	"context"
	"os"
	"strconv"
	"time"

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

func vcodeKey(scene, target string) string  { return vcodeKeyPrefix + scene + ":" + target }
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
