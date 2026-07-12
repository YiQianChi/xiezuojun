// extractor — 人物设定角色提取器
// 异步调用 AI 从项目所有人物设定文档提取角色，写入角色卡文件

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"xiezuojun-server/internal/ai"
)

// memoryModel returns the model name for memory extraction AI calls.
// ponytail: env var fallback chain, no new deps
func memoryModel() string {
	if m := os.Getenv("MEMORY_MODEL"); m != "" {
		return m
	}
	if m := os.Getenv("AI_MODEL"); m != "" {
		return m
	}
	return "MiniMax-M3"
}

// ExtractCharacters 从项目所有人物设定文档拼接的全文中提取角色
func ExtractCharacters(ctx context.Context, aiClient *ai.Client, storage *MemoryStorage, uid, pid, content string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[memory] ExtractCharacters panicked: %v", r)
		}
	}()

	if content == "" {
		log.Printf("[memory] empty content, skip")
		return
	}

	extractPrompt := `分析以下小说人物设定文档，提取所有角色信息，以JSON格式返回：
{
  "characters": [{"name": "角色名", "role": "主角/配角/路人", "description": "简短描述"}]
}
只返回JSON，不要其他内容。如果没有角色信息，返回空数组。`

	extractResult, err := aiClient.Complete(ctx, memoryModel(), []ai.Message{
		{Role: "system", Content: extractPrompt},
		{Role: "user", Content: content},
	})
	if err != nil {
		log.Printf("[memory] failed to extract characters: %v", err)
		return
	}

	var extracted struct {
		Characters []struct {
			Name        string `json:"name"`
			Role        string `json:"role"`
			Description string `json:"description"`
		} `json:"characters"`
	}

	// 尝试提取 JSON（可能被 ```json 包裹）
	jsonStr := extractResult
	if idx := strings.Index(jsonStr, "```json"); idx >= 0 {
		jsonStr = jsonStr[idx+7:]
		if endIdx := strings.Index(jsonStr, "```"); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx]
		}
	}
	if idx := strings.Index(jsonStr, "```"); idx >= 0 {
		jsonStr = jsonStr[idx+3:]
		if endIdx := strings.Index(jsonStr, "```"); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx]
		}
	}
	jsonStr = strings.TrimSpace(jsonStr)

	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		log.Printf("[memory] failed to parse extract JSON: %v", err)
		return
	}

	for _, char := range extracted.Characters {
		if char.Name == "" {
			continue
		}
		existing, err := storage.ReadCharacter(uid, pid, char.Name)
		if err == nil && existing != "" {
			log.Printf("[memory] character %s already exists, skip", char.Name)
			continue
		}

		card := fmt.Sprintf(`# %s

> auto_generated: true

## 基本信息
- 身份：%s

## 描述
%s

## 备注
由系统自动识别创建，请编辑完善。
`, char.Name, char.Role, char.Description)

		if err := storage.WriteCharacter(uid, pid, char.Name, card); err != nil {
			log.Printf("[memory] failed to create character card %s: %v", char.Name, err)
		} else {
			log.Printf("[memory] auto-created character card: %s", char.Name)
		}
	}
}

// DetectCharacterNames 从文本中检测疑似角色名
// 规则：连续2-4个中文字符出现3次以上 → 疑似角色名
func DetectCharacterNames(text string) []string {
	nameCount := make(map[string]int)
	runes := []rune(text)

	// 遍历所有 2-4 字中文片段
	for length := 2; length <= 4; length++ {
		for i := 0; i+length <= len(runes); i++ {
			segment := string(runes[i : i+length])
			if isChineseName(segment) {
				nameCount[segment]++
			}
		}
	}

	// 过滤出现 3 次以上的
	var names []string
	for name, count := range nameCount {
		if count >= 3 {
			names = append(names, name)
		}
	}

	// 去重：如果 "张三" 和 "张三丰" 都存在，保留短的（更可能是独立名字）
	// 除非长的是独立的
	names = deduplicateNames(names)
	return names
}

// isChineseName 判断是否是合法的中文名片段
func isChineseName(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return utf8.RuneCountInString(s) >= 2
}

// deduplicateNames 去重：如果短名是长名的子串且计数差异不大，保留短名
func deduplicateNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	// 按长度排序（短的优先）
	// 简单策略：过滤掉那些是其他名字子串的，除非它本身就是最常见的
	result := make([]string, 0)
	for _, name := range names {
		isSubstring := false
		for _, other := range names {
			if other != name && len(other) > len(name) && strings.Contains(other, name) {
				isSubstring = true
				break
			}
		}
		if !isSubstring {
			result = append(result, name)
		}
	}

	return result
}
