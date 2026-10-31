package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SelfCheck AI 味自检（右栏「自检」页签）：输出对标炼笔司——指数 + 归类证据 + 改写建议。
// 非流式 Complete（推理模型首字慢，全量 30-90s，前端 axios 已放宽超时）。

type selfCheckReq struct {
	Text string `json:"text"`
}

type selfCheckItem struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"` // high | mid | low
	Sentence   string `json:"sentence"`
	Suggestion string `json:"suggestion"`
}

type selfCheckResult struct {
	Score  int             `json:"score"`  // 0-100，越高越"AI 味"
	Level  string          `json:"level"`  // 低 | 中 | 高
	Summary string         `json:"summary"`
	Items  []selfCheckItem `json:"items"`
}

// RegisterSelfCheck 注册自检路由（挂在 authd 组；model 用第一个默认模型）
func RegisterSelfCheck(rg *gin.RouterGroup, client *Client, model string) {
	rg.POST("/ai/selfcheck", func(c *gin.Context) {
		var req selfCheckReq
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Text) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先写点正文再自检"})
			return
		}
		text := strings.TrimSpace(req.Text)
		if len([]rune(text)) > 6000 {
			text = string([]rune(text)[:6000])
		}

		prompt := fmt.Sprintf(`你是中文网文编辑。检查下面这段正文里的"AI 味"问题。AI 味指：词藻堆砌（仿佛/宛如/宛如晨曦类滥用比喻）、空洞副词（忽然/竟然/莫名）、口水对话（无信息量的对话铺垫）、总结式抒情（"这一刻他明白了…"）、句式雷同（连续排比/统一长句）。

只输出一个 JSON 对象，不要任何其他文字：
{"score":0到100的整数,"level":"低|中|高","summary":"一句话总评","items":[{"type":"问题类型(词藻堆砌|空洞副词|口水对话|总结式抒情|句式雷同)","severity":"high|mid|low","sentence":"原文中命中的那句","suggestion":"具体的改写建议或改写后的句子"}]}

最多列 6 条，按严重程度降序。没有问题则 items 为空数组、score 给低分。

正文：
%s`, text)

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
		defer cancel()
		out, err := client.Complete(ctx, model, []Message{
			{Role: "user", Content: prompt},
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "AI 暂时不可用，请稍后再试"})
			return
		}

		res := parseSelfCheckJSON(out)
		if res == nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "AI 返回格式异常，请重试"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": res})
	})
}

// parseSelfCheckJSON 从模型输出里抠 JSON（容忍 ```json 围栏和前后废话）
func parseSelfCheckJSON(s string) *selfCheckResult {
	s = strings.TrimSpace(s)
	// 围栏
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		if j := strings.Index(s, "\n"); j >= 0 {
			s = s[j+1:]
		}
		if k := strings.LastIndex(s, "```"); k >= 0 {
			s = s[:k]
		}
	}
	// 首尾大括号
	st := strings.Index(s, "{")
	en := strings.LastIndex(s, "}")
	if st < 0 || en <= st {
		return nil
	}
	var res selfCheckResult
	if err := json.Unmarshal([]byte(s[st:en+1]), &res); err != nil {
		return nil
	}
	if res.Score < 0 {
		res.Score = 0
	}
	if res.Score > 100 {
		res.Score = 100
	}
	if res.Level == "" {
		switch {
		case res.Score >= 60:
			res.Level = "高"
		case res.Score >= 30:
			res.Level = "中"
		default:
			res.Level = "低"
		}
	}
	if res.Items == nil {
		res.Items = []selfCheckItem{}
	}
	if len(res.Items) > 6 {
		res.Items = res.Items[:6]
	}
	// 模型幻觉兜底：正则确认 sentence 确实出现在原文外层提示里无法做（原文没传回）——信任度靠 prompt 约束
	return &res
}
