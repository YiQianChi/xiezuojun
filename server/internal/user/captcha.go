package user

// 点选验证码：服务端渲染 PNG，图上散布 ★ ▲ ◆ 三个图形，用户按提示顺序点击。
// 坐标只存服务端（图片是位图，文本读不出），脚本须做图像识别；配合配额足够防刷。
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
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const captchaPrefix = "xzj:captcha:"
const clickTol = 24 // 命中半径 px

type clickChallenge struct {
	ID   string `json:"id"`
	Img  string `json:"img"` // PNG data URI（含三个图形）
	Hint string `json:"hint"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

var captchaColors = [][2]string{
	{"#c7d2fe", "#e0e7ff"}, {"#fde68a", "#fef3c7"}, {"#bbf7d0", "#dcfce7"},
	{"#fbcfe8", "#fce7f3"}, {"#bfdbfe", "#dbeafe"},
}

// 三个图形：固定点击顺序 star → triangle → diamond；随机旋转 + 伪立体着色
var clickShapes = []struct {
	glyph string
	draw  func(img *image.RGBA, cx, cy, r int, ang float64, c color.RGBA)
}{
	{"★", drawStar},
	{"▲", drawTriangle},
	{"◆", drawDiamond},
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

// ---- 画图（纯标准库像素操作）----

func hexToRGB(s string) color.RGBA {
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// fillShape 伪立体着色填充：inside 判定（dx,dy 已按 ang 反旋转），亮度沿光照方向渐变
func fillShape(img *image.RGBA, cx, cy, r int, ang float64, c color.RGBA, inside func(dx, dy float64) bool) {
	ca, sa := math.Cos(ang), math.Sin(ang)
	for py := cy - r - 3; py <= cy+r+3; py++ {
		for px := cx - r - 3; px <= cx+r+3; px++ {
			if px < 0 || py < 0 || px >= img.Rect.Dx() || py >= img.Rect.Dy() {
				continue
			}
			dx, dy := px-cx, py-cy
			// 反旋转到图形本地坐标
			lx := float64(dx)*ca + float64(dy)*sa
			ly := -float64(dx)*sa + float64(dy)*ca
			if !inside(lx, ly) {
				continue
			}
			// 伪立体：高光在左上，右下压暗
			d := math.Hypot(float64(dx), float64(dy)) / float64(r)
			light := 1.0 + 0.30*(1-d) - 0.18*d - 0.22*(float64(dx)+float64(dy))/(2*float64(r))
			shade := func(v uint8) uint8 {
				n := int(float64(v) * light)
				if n < 0 {
					return 0
				}
				if n > 255 {
					return 255
				}
				return uint8(n)
			}
			base := img.RGBAAt(px, py)
			a := int(c.A)
			rr, gg, bb := shade(c.R), shade(c.G), shade(c.B)
			img.SetRGBA(px, py, color.RGBA{
				uint8((int(rr)*a + int(base.R)*(255-a)) / 255),
				uint8((int(gg)*a + int(base.G)*(255-a)) / 255),
				uint8((int(bb)*a + int(base.B)*(255-a)) / 255),
				255,
			})
		}
	}
}

func drawStar(img *image.RGBA, cx, cy, r int, ang float64, c color.RGBA) {
	pts := make([][2]float64, 10)
	for i := 0; i < 10; i++ {
		a := -math.Pi/2 + float64(i)*math.Pi/5
		rad := float64(r)
		if i%2 == 1 {
			rad = float64(r) * 0.45
		}
		pts[i] = [2]float64{rad * math.Cos(a), rad * math.Sin(a)}
	}
	fillShape(img, cx, cy, r, ang, c, func(dx, dy float64) bool {
		return pointInPoly(dx, dy, pts)
	})
}

func pointInPoly(x, y float64, pts [][2]float64) bool {
	in := false
	j := len(pts) - 1
	for i := 0; i < len(pts); i++ {
		if (pts[i][1] > y) != (pts[j][1] > y) &&
			x < (pts[j][0]-pts[i][0])*(y-pts[i][1])/(pts[j][1]-pts[i][1])+pts[i][0] {
			in = !in
		}
		j = i
	}
	return in
}

func drawTriangle(img *image.RGBA, cx, cy, r int, ang float64, c color.RGBA) {
	pts := [][2]float64{
		{0, -float64(r)}, {float64(r) * 0.87, float64(r) / 2}, {-float64(r) * 0.87, float64(r) / 2},
	}
	fillShape(img, cx, cy, r, ang, c, func(dx, dy float64) bool {
		return pointInPoly(dx, dy, pts)
	})
}

func drawDiamond(img *image.RGBA, cx, cy, r int, ang float64, c color.RGBA) {
	fillShape(img, cx, cy, r, ang, c, func(dx, dy float64) bool {
		return math.Abs(dx)+math.Abs(dy) <= float64(r)
	})
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	fillShape(img, cx, cy, r, 0, c, func(dx, dy float64) bool { return dx*dx+dy*dy <= float64(r)*float64(r) })
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// newClickCaptcha 出题：随机底图 + 三个不重叠图形；答案（顺序坐标）存 Redis 5 分钟
func newClickCaptcha() clickChallenge {
	const W, H, R = 280, 150, 20
	c := captchaColors[randIntn(len(captchaColors))]
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	c0, c1 := hexToRGB(c[0]), hexToRGB(c[1])
	for x := 0; x < W; x++ {
		t := float64(x) / float64(W)
		for py := 0; py < H; py++ {
			img.SetRGBA(x, py, color.RGBA{
				uint8(float64(c0.R) + (float64(c1.R)-float64(c0.R))*t),
				uint8(float64(c0.G) + (float64(c1.G)-float64(c0.G))*t),
				uint8(float64(c0.B) + (float64(c1.B)-float64(c0.B))*t),
				255,
			})
		}
	}
	for i := 0; i < 5; i++ { // 噪声圆
		drawCircle(img, randIntn(W), randIntn(H), 8+randIntn(18), color.RGBA{255, 255, 255, 90})
	}
	colors := []color.RGBA{{220, 38, 38, 235}, {37, 99, 235, 235}, {22, 163, 74, 235}}
	pos := [3][2]int{}
	angs := [3]float64{}
	for i := 0; i < 3; i++ {
		angs[i] = float64(randIntn(628)) / 100 // 0~2π 随机旋转
	}
	for i := 0; i < 3; i++ {
		// 找一个与已放图形不重叠的位置（中心距 > 2.6R）
		for try := 0; ; try++ {
			x, y := 30+randIntn(W-60), 30+randIntn(H-60)
			ok := true
			for j := 0; j < i; j++ {
				if abs(x-pos[j][0])+abs(y-pos[j][1]) < 70 {
					ok = false
					break
				}
			}
			if ok || try > 50 {
				pos[i] = [2]int{x, y}
				break
			}
		}
		drawCircle(img, pos[i][0], pos[i][1], R+5, color.RGBA{255, 255, 255, 160}) // 白底衬
		clickShapes[i].draw(img, pos[i][0], pos[i][1], R, angs[i], colors[i])
	}
	buf := &bytes.Buffer{}
	png.Encode(buf, img)
	id := uuid.NewString()
	ans, _ := json.Marshal(pos)
	vcodeRdb.Set(vcodeCtx, captchaPrefix+id, string(ans), 5*time.Minute)
	hint := ""
	for i, s := range clickShapes {
		if i > 0 {
			hint += " → "
		}
		hint += s.glyph
	}
	return clickChallenge{ID: id, Img: "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), Hint: hint, W: W, H: H}
}

// clickAnswer 客户端提交：按顺序的点击坐标 + 总耗时 ms
type clickAnswer struct {
	Points [][2]int `json:"points"`
	Dur    int      `json:"dur"`
}

// verifyCaptcha 校验并销毁（一次性）：3 个点按顺序各自命中 24px 半径，总时长 400ms~30s
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
	var target [][2]int
	if err := json.Unmarshal([]byte(want), &target); err != nil || len(target) != 3 {
		return false
	}
	var ans clickAnswer
	if err := json.Unmarshal([]byte(answerJSON), &ans); err != nil {
		log.Printf("[captcha] reject: bad json (%d bytes)", len(answerJSON))
		return false
	}
	if len(ans.Points) != 3 {
		log.Printf("[captcha] reject: points=%d", len(ans.Points))
		return false
	}
	for i, p := range ans.Points {
		if abs(p[0]-target[i][0]) > clickTol || abs(p[1]-target[i][1]) > clickTol {
			log.Printf("[captcha] reject: point%d (%d,%d) 目标(%d,%d)", i+1, p[0], p[1], target[i][0], target[i][1])
			return false
		}
	}
	if ans.Dur < 400 || ans.Dur > 30000 {
		log.Printf("[captcha] reject: dur=%dms", ans.Dur)
		return false
	}
	return true
}
