package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 中文音色列表（Google Translate TTS 不支持选音色，但保留列表让前端展示）
var VoiceList = []gin.H{
	{"name": "中文女声（默认）", "voice": "zh-CN", "gender": "Female"},
	{"name": "中文男声", "voice": "zh-CN-male", "gender": "Male"},
}

// HandleTTS 朗读接口：POST /api/tts
// body: { "text": "要朗读的文字", "voice": "zh-CN" }
// 返回 audio/mp3 二进制
func HandleTTS(c *gin.Context) {
	var req struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if len(req.Text) == 0 {
		c.JSON(400, gin.H{"error": "text is required"})
		return
	}
	if len(req.Text) > 5000 {
		req.Text = req.Text[:5000]
	}

	audio, err := synthesizeGoogle(req.Text)
	if err != nil {
		c.JSON(500, gin.H{"error": "TTS failed: " + err.Error()})
		return
	}

	c.Data(200, "audio/mpeg", audio)
}

// HandleVoices 返回可用音色列表
func HandleVoices(c *gin.Context) {
	c.JSON(200, gin.H{"data": VoiceList})
}

// synthesizeGoogle 使用 Google Translate TTS 合成语音
// 限制：单次最多 ~200 字符，需要分段请求再拼接
func synthesizeGoogle(text string) ([]byte, error) {
	// 按句子分割（中文句号、问号、感叹号、换行）
	chunks := splitText(text, 180)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no text to synthesize")
	}

	var audioBuf bytes.Buffer
	client := &http.Client{Timeout: 15 * time.Second}

	for i, chunk := range chunks {
		u := fmt.Sprintf("https://translate.google.com/translate_tts?ie=UTF-8&q=%s&tl=zh-CN&client=tw-ob", url.QueryEscape(chunk))
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("request %d: %w", i, err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch %d: %w", i, err)
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("fetch %d: status=%d body=%s", i, resp.StatusCode, string(body))
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read %d: %w", i, err)
		}
		audioBuf.Write(data)
	}

	if audioBuf.Len() == 0 {
		return nil, fmt.Errorf("no audio data received")
	}

	return audioBuf.Bytes(), nil
}

// splitText 将长文本分割成不超过 maxLen 的片段
// 按自然标点优先分割
func splitText(text string, maxLen int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// 先按句号/问号/感叹号/换行分句
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '。' || r == '？' || r == '！' || r == '.' || r == '?' || r == '!' || r == '\n' || r == '；' || r == ';'
	})

	var chunks []string
	var current strings.Builder

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// 如果当前片段 + 新句子不超长，拼接
		if current.Len()+len(s)+1 <= maxLen {
			if current.Len() > 0 {
				current.WriteString("。")
			}
			current.WriteString(s)
		} else {
			// 先保存当前片段
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			// 如果单个句子就超长，强制按 maxLen 截断
			if len(s) > maxLen {
				for len(s) > maxLen {
					chunks = append(chunks, s[:maxLen])
					s = s[maxLen:]
				}
				if s != "" {
					current.WriteString(s)
				}
			} else {
				current.WriteString(s)
			}
		}
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// avoid unused import
var _ = json.Marshal
