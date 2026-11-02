package user

// 渐进式数学验证码：手机号/IP 当日发送达到阈值后，发短信前必须先过验证码。
// 正常用户一天 1~2 条短信，永远见不到验证码；刷量的从第 3 条起每条都要人机交互。

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const captchaPrefix = "xzj:captcha:"

// smsNeedsCaptcha 当日该手机号或 IP 已发过 >= threshold 条则要求验证码
func smsNeedsCaptcha(phone, ip string) bool {
	const threshold = 2
	d := time.Now().Format("20060102")
	for _, k := range []string{smsQuotaKey("phone", phone, d), smsQuotaKey("ip", ip, d)} {
		if v, err := vcodeRdb.Get(vcodeCtx, k).Result(); err == nil {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= threshold {
				return true
			}
		}
	}
	return false
}

// newCaptcha 出一道加减题（结果保证 0~9），答案存 Redis 5 分钟
func newCaptcha() (id, question string) {
	ri := func(max int) int { n, _ := rand.Int(rand.Reader, big.NewInt(int64(max))); return int(n.Int64()) }
	a, b := ri(9)+1, ri(9)+1 // 1~9
	if ri(2) == 0 {
		question, id = fmt.Sprintf("%d + %d = ?", a, b), uuid.NewString()
		vcodeRdb.Set(vcodeCtx, captchaPrefix+id, strconv.Itoa(a+b), 5*time.Minute)
	} else {
		question, id = fmt.Sprintf("%d - %d = ?", a+b, a), uuid.NewString()
		vcodeRdb.Set(vcodeCtx, captchaPrefix+id, strconv.Itoa(b), 5*time.Minute)
	}
	return id, question
}

// verifyCaptcha 校验并销毁（一次性）；答案错误可立即换题重试
func verifyCaptcha(id, answer string) bool {
	if id == "" {
		return false
	}
	key := captchaPrefix + id
	want, err := vcodeRdb.Get(vcodeCtx, key).Result()
	if err != nil {
		return false
	}
	vcodeRdb.Del(vcodeCtx, key)
	return strings.TrimSpace(answer) == want
}
