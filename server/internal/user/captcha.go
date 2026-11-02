package user

// 滑块验证码：服务端生成缺口位置，客户端拖动拼图对准，校验落点容差 + 拖动轨迹（采样数/时长）。
// 相比明文算术题，答案（缺口 x 坐标）不出服务端，脚本必须做图像分析或伪造人手轨迹；
// 配合单号/单IP/全站配额，足够防刷短信。
// 渐进式：手机号/IP 当日发送达到阈值（smsNeedsCaptcha）后才触发，正常用户一天 1~2 条无感。

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const captchaPrefix = "xzj:captcha:"

type sliderChallenge struct {
	ID    string `json:"id"`
	Bg    string `json:"bg"` // 整张底图 SVG（含缺口）
	Piece string `json:"piece"`
	Y     int    `json:"y"` // 拼图块 top（px）
	W     int    `json:"w"` // 底图宽
}

var captchaColors = [][2]string{
	{"#c7d2fe", "#e0e7ff"}, {"#fde68a", "#fef3c7"}, {"#bbf7d0", "#dcfce7"},
	{"#fbcfe8", "#fce7f3"}, {"#bfdbfe", "#dbeafe"},
}

func randIntn(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
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

// newSliderCaptcha 出题：随机底图 + 缺口，答案（缺口圆心 x）只存 Redis（5 分钟）
func newSliderCaptcha() sliderChallenge {
	const W, H, R = 280, 150, 24
	target := 70 + randIntn(W-140) // 缺口圆心 x
	y := R + 10 + randIntn(H-2*R-20)
	c := captchaColors[randIntn(len(captchaColors))]
	shapes := ""
	for i := 0; i < 4; i++ {
		shapes += fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="rgba(255,255,255,0.35)"/>`,
			randIntn(W), randIntn(H), 8+randIntn(20))
	}
	bg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+
		`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`+
		`<rect width="%d" height="%d" fill="url(#g)"/>%s`+
		`<circle cx="%d" cy="%d" r="%d" fill="rgba(0,0,0,0.35)" stroke="#fff" stroke-width="2"/></svg>`,
		W, H, W, H, c[0], c[1], W, H, shapes, target, y, R)
	piece := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+
		`<circle cx="%d" cy="%d" r="%d" fill="rgba(255,255,255,0.8)" stroke="rgba(0,0,0,0.35)" stroke-width="2"/></svg>`,
		2*R, 2*R, R, R, R)
	id := uuid.NewString()
	vcodeRdb.Set(vcodeCtx, captchaPrefix+id, strconv.Itoa(target), 5*time.Minute)
	return sliderChallenge{ID: id, Bg: bg, Piece: piece, Y: y - R, W: W}
}

// captchaAnswer 客户端提交：落点（拼图圆心 x）+ 拖动轨迹 [ [ms偏移, x], ... ]
type captchaAnswer struct {
	X     int      `json:"x"`
	Trail [][2]int `json:"trail"`
}

// verifyCaptcha 校验并销毁（一次性）：落点容差 8px，且轨迹像人（≥5 采样、200ms~20s）
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
	target, err := strconv.Atoi(want)
	if err != nil {
		return false
	}
	var ans captchaAnswer
	if err := json.Unmarshal([]byte(answerJSON), &ans); err != nil {
		return false
	}
	if ans.X < target-8 || ans.X > target+8 {
		return false
	}
	if len(ans.Trail) < 5 {
		return false
	}
	dur := ans.Trail[len(ans.Trail)-1][0] - ans.Trail[0][0]
	if dur < 200 || dur > 20000 {
		return false
	}
	return true
}
