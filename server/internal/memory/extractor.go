// extractor — 章节提取器
// 异步调用 AI 从章节内容提取摘要、角色、事件等，写入记忆文件系统

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

// ExtractAndSaveSummary 从章节内容提取摘要、角色、事件等并保存
// 此函数会被 goroutine 异步调用，内部 recover panic
func ExtractAndSaveSummary(ctx context.Context, aiClient *ai.Client, storage *MemoryStorage, uid, pid, docID, content string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[memory] ExtractAndSaveSummary panicked for doc %s: %v", docID, r)
		}
	}()

	if content == "" {
		log.Printf("[memory] empty content for doc %s, skip", docID)
		return
	}

	// 1. 生成章节摘要
	summary, err := aiClient.Complete(ctx, memoryModel(), []ai.Message{
		{Role: "system", Content: "你是一个专业的小说章节摘要生成器。请用简洁的Markdown格式总结以下章节的核心情节，不超过500字。包含：主要事件、人物互动、关键转折。"},
		{Role: "user", Content: content},
	})
	if err != nil {
		log.Printf("[memory] failed to generate summary for doc %s: %v", docID, err)
	} else {
		if err := storage.WriteChapterSummary(uid, pid, docID, summary); err != nil {
			log.Printf("[memory] failed to save summary for doc %s: %v", docID, err)
		} else {
			log.Printf("[memory] saved summary for doc %s", docID)
		}
	}

	// 2. 提取角色、事件、地点、伏笔（一次调用，JSON返回）
	extractPrompt := `分析以下小说章节内容，提取以下信息，以JSON格式返回：
{
  "characters": [{"name": "角色名", "role": "主角/配角/路人", "description": "简短描述"}],
  "events": ["事件1", "事件2"],
  "locations": ["地点1", "地点2"],
  "foreshadowing": ["伏笔1", "伏笔2"]
}
只返回JSON，不要其他内容。如果没有某类信息，返回空数组。`

	extractResult, err := aiClient.Complete(ctx, memoryModel(), []ai.Message{
		{Role: "system", Content: extractPrompt},
		{Role: "user", Content: content},
	})
	if err != nil {
		log.Printf("[memory] failed to extract entities for doc %s: %v", docID, err)
		return
	}

	// 3. 解析 JSON 结果
	var extracted struct {
		Characters   []struct {
			Name        string `json:"name"`
			Role        string `json:"role"`
			Description string `json:"description"`
		} `json:"characters"`
		Events       []string `json:"events"`
		Locations    []string `json:"locations"`
		Foreshadowing []string `json:"foreshadowing"`
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
		log.Printf("[memory] failed to parse extract JSON for doc %s: %v", docID, err)
		return
	}

	// 4. 新角色自动创建骨架角色卡
	for _, char := range extracted.Characters {
		if char.Name == "" {
			continue
		}
		// 检查是否已存在
		existing, err := storage.ReadCharacter(uid, pid, char.Name)
		if err == nil && existing != "" {
			log.Printf("[memory] character %s already exists, skip", char.Name)
			continue
		}

		// 创建骨架角色卡
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

	// 5. 追加到 TIMELINE.md
	if len(extracted.Events) > 0 {
		timelineEntry := fmt.Sprintf("\n## %s\n", docID)
		for _, evt := range extracted.Events {
			timelineEntry += fmt.Sprintf("- %s\n", evt)
		}

		existing, _ := storage.ReadMemoryFile(uid, pid, "TIMELINE.md")
		storage.WriteMemoryFile(uid, pid, "TIMELINE.md", existing+timelineEntry)
		log.Printf("[memory] appended to TIMELINE.md for doc %s", docID)
	}

	// 6. 追加到 HOOKS.md
	if len(extracted.Foreshadowing) > 0 {
		hoooksEntry := fmt.Sprintf("\n## %s\n", docID)
		for _, hook := range extracted.Foreshadowing {
			hoooksEntry += fmt.Sprintf("- [ ] %s\n", hook)
		}

		existing, _ := storage.ReadMemoryFile(uid, pid, "HOOKS.md")
		storage.WriteMemoryFile(uid, pid, "HOOKS.md", existing+hoooksEntry)
		log.Printf("[memory] appended to HOOKS.md for doc %s", docID)
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
