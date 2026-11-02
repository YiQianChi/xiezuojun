package user

// 滑块验证码：服务端生成缺口位置，客户端拖动拼图对准，校验落点容差 + 拖动轨迹（采样数/时长）。
// 相比明文算术题，答案（缺口 x 坐标）不出服务端，脚本必须做图像分析或伪造人手轨迹；
// 配合单号/单IP/全站配额，足够防刷短信。
// 渐进式：手机号/IP 当日发送达到阈值（smsNeedsCaptcha）后才触发，正常用户一天 1~2 条无感。

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
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

// newSliderCaptcha 出题：服务端渲染 PNG 底图（缺口坐标不可从文本读出），答案只存 Redis（5 分钟）
func newSliderCaptcha() sliderChallenge {
	const W, H, R = 280, 150, 24
	target := 70 + randIntn(W-140) // 缺口圆心 x
	y := R + 10 + randIntn(H-2*R-20)
	c := captchaColors[randIntn(len(captchaColors))]
	bg := renderSliderBg(W, H, target, y, c)
	piece := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+
		`<circle cx="%d" cy="%d" r="%d" fill="rgba(255,255,255,0.8)" stroke="rgba(0,0,0,0.35)" stroke-width="2"/></svg>`,
		2*R, 2*R, R, R, R)
	id := uuid.NewString()
	vcodeRdb.Set(vcodeCtx, captchaPrefix+id, strconv.Itoa(target), 5*time.Minute)
	return sliderChallenge{ID: id, Bg: bg, Piece: piece, Y: y - R, W: W}
}

// hexToRGB "#rrggbb" → color.RGBA
func hexToRGB(s string) color.RGBA {
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// fillCircle 以 alpha 混合画圆
func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for py := cy - r; py <= cy+r; py++ {
		for px := cx - r; px <= cx+r; px++ {
			if px < 0 || py < 0 || px >= img.Rect.Dx() || py >= img.Rect.Dy() {
				continue
			}
			dx, dy := px-cx, py-cy
			if dx*dx+dy*dy > r*r {
				continue
			}
			base := img.RGBAAt(px, py)
			a := int(c.A)
			img.SetRGBA(px, py, color.RGBA{
				uint8((int(c.R)*a + int(base.R)*(255-a)) / 255),
				uint8((int(c.G)*a + int(base.G)*(255-a)) / 255),
				uint8((int(c.B)*a + int(base.B)*(255-a)) / 255),
				255,
			})
		}
	}
}

// renderSliderBg 纯标准库画底图：渐变 + 随机噪声圆 + 缺口（含白色描边），输出 data URI
func renderSliderBg(w, h, target, y int, c [2]string) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c0, c1 := hexToRGB(c[0]), hexToRGB(c[1])
	for x := 0; x < w; x++ {
		t := float64(x) / float64(w)
		for py := 0; py < h; py++ {
			img.SetRGBA(x, py, color.RGBA{
				uint8(float64(c0.R) + (float64(c1.R)-float64(c0.R))*t),
				uint8(float64(c0.G) + (float64(c1.G)-float64(c0.G))*t),
				uint8(float64(c0.B) + (float64(c1.B)-float64(c0.B))*t),
				255,
			})
		}
	}
	for i := 0; i < 4; i++ { // 噪声圆（半透明白）
		fillCircle(img, randIntn(w), randIntn(h), 8+randIntn(20), color.RGBA{255, 255, 255, 90})
	}
	fillCircle(img, target, y, 24, color.RGBA{0, 0, 0, 110}) // 缺口
	for px := target - 25; px <= target+25; px++ {           // 白色描边环
		for py := y - 25; py <= y+25; py++ {
			if px < 0 || py < 0 || px >= w || py >= h {
				continue
			}
			dx, dy := px-target, py-y
			d2 := dx*dx + dy*dy
			if d2 <= 25*25 && d2 >= 22*22 {
				img.SetRGBA(px, py, color.RGBA{255, 255, 255, 200})
			}
		}
	}
	buf := &bytes.Buffer{}
	png.Encode(buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// captchaAnswer 客户端提交：落点（拼图圆心 x）+ 拖动轨迹 [ [ms偏移, x], ... ]
type captchaAnswer struct {
	X     int      `json:"x"`
	Trail [][2]int `json:"trail"`
}

// verifyCaptcha 校验并销毁（一次性）：落点容差 12px，且轨迹基本像人（≥3 采样、120ms~20s）
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
		log.Printf("[captcha] reject: bad json (%d bytes)", len(answerJSON))
		return false
	}
	if d := ans.X - target; d < -15 || d > 15 {
		log.Printf("[captcha] reject: x=%d target=%d 偏差=%dpx samples=%d", ans.X, target, d, len(ans.Trail))
		return false
	}
	if len(ans.Trail) < 3 {
		log.Printf("[captcha] reject: samples=%d (x=%d target=%d)", len(ans.Trail), ans.X, target)
		return false
	}
	dur := ans.Trail[len(ans.Trail)-1][0] - ans.Trail[0][0]
	if dur < 100 || dur > 20000 {
		log.Printf("[captcha] reject: dur=%dms samples=%d", dur, len(ans.Trail))
		return false
	}
	return true
}
