package user

// 点选验证码（GoCaptcha 开源库）：照片背景 + 书法汉字，按提示条顺序点击。
// 答案（点阵坐标）只存服务端 Redis；校验用官方 click.Validate + 人手时序。
// 渐进式：手机号/IP 当日发送达到阈值（smsNeedsCaptcha）后才触发，正常用户一天 1~2 条无感。

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	"github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha/v2/base/option"
	goclick "github.com/wenlng/go-captcha/v2/click"
)

const captchaPrefix = "xzj:captcha:"

type clickChallenge struct {
	ID    string `json:"id"`
	Img   string `json:"img"`   // 大图 data URI（照片背景+汉字）
	Thumb string `json:"thumb"` // 提示条 data URI（按此顺序点击）
	N     int    `json:"n"`     // 需要点的个数（=提示条字数）
	W     int    `json:"w"`
	H     int    `json:"h"`
}

var (
	captOnce sync.Once
	captInst goclick.Captcha
)

// getGoCaptcha 单例初始化（字体+背景图打包在素材库 bindata 里，无外部文件依赖）
func getGoCaptcha() goclick.Captcha {
	captOnce.Do(func() {
		fonts, err := fzshengsksjw.GetFont()
		if err != nil {
			panic("captcha font: " + err.Error())
		}
		imgs, err := imagesv2.GetImages()
		if err != nil {
			panic("captcha backgrounds: " + err.Error())
		}
		b := goclick.NewBuilder(
			goclick.WithRangeLen(option.RangeVal{Min: 4, Max: 6}),       // 图上散布 4~6 个字
			goclick.WithRangeVerifyLen(option.RangeVal{Min: 2, Max: 4}), // 需按序点 2~4 个
		)
		b.SetResources(
			goclick.WithChars(chars.GetChineseChars()),
			goclick.WithFonts([]*truetype.Font{fonts}),
			goclick.WithBackgrounds(imgs),
		)
		captInst = b.Make()
	})
	return captInst
}

// newClickCaptcha 出题；点阵答案存 Redis 5 分钟
func newClickCaptcha() clickChallenge {
	data, err := getGoCaptcha().Generate()
	if err != nil {
		log.Printf("[captcha] generate failed: %v", err)
		return clickChallenge{}
	}
	img, _ := data.GetMasterImage().ToBase64()
	thumb, _ := data.GetThumbImage().ToBase64()
	dots := data.GetData()
	ans, _ := json.Marshal(dots)
	id := uuid.NewString()
	vcodeRdb.Set(vcodeCtx, captchaPrefix+id, string(ans), 5*time.Minute)
	return clickChallenge{ID: id, Img: img, Thumb: thumb, N: len(dots), W: 300, H: 220}
}

// clickAnswer 客户端提交：按提示顺序的点击坐标 + 总耗时 ms
type clickAnswer struct {
	Points [][2]int `json:"points"`
	Dur    int      `json:"dur"`
}

// verifyCaptcha 校验并销毁（一次性）：逐点 click.Validate（官方算法）+ 人手时序
func verifyCaptcha(id, answerJSON string) bool {
	if id == "" {
		return false
	}
	key := captchaPrefix + id
	want, err := vcodeRdb.Get(vcodeCtx, key).Result()
	if err != nil {
		return false
	}
	vcodeRdb.Del(vcodeCtx, key)
	var dots map[int]*goclick.Dot
	if err := json.Unmarshal([]byte(want), &dots); err != nil || len(dots) == 0 {
		return false
	}
	var ans clickAnswer
	if err := json.Unmarshal([]byte(answerJSON), &ans); err != nil {
		log.Printf("[captcha] reject: bad json (%d bytes)", len(answerJSON))
		return false
	}
	if len(ans.Points) != len(dots) {
		log.Printf("[captcha] reject: points=%d need=%d", len(ans.Points), len(dots))
		return false
	}
	for i, p := range ans.Points {
		dot := dots[i]
		if !goclick.Validate(p[0], p[1], dot.X, dot.Y, dot.Width, dot.Height, 5) {
			log.Printf("[captcha] reject: point%d (%d,%d) 目标(%d,%d)", i+1, p[0], p[1], dot.X, dot.Y)
			return false
		}
	}
	if ans.Dur < 300 || ans.Dur > 30000 {
		log.Printf("[captcha] reject: dur=%dms", ans.Dur)
		return false
	}
	return true
}

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
