package project

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"xiezuojun-server/internal/model"
)

const (
	defaultSearchLimit = 50
	maxSearchLimit     = 100
	maxSearchDocs      = 500
)

type documentSearchMatch struct {
	DocID           uint   `json:"doc_id"`
	Title           string `json:"title"`
	Line            int    `json:"line"`
	Col             int    `json:"col"`
	Snippet         string `json:"snippet"`
	OccurrenceIndex int    `json:"occurrence_index"`
}

type documentSearchResponse struct {
	Items   []documentSearchMatch `json:"items"`
	Total   int                   `json:"total"`
	Limit   int                   `json:"limit"`
	HasMore bool                  `json:"has_more"`
}

// SearchDocuments 在当前项目内全文搜索章节正文。
// 正文权威源仍是文件，documents.search_text 只是轻量搜索缓存。
func (h *Handler) SearchDocuments(c *gin.Context) {
	userIDValue, _ := c.Get("userID")
	userID, _ := userIDValue.(string)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	if !h.requireProjectOwner(c, uint(projectID)) {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if !validSearchQuery(query) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入至少 2 个中文字符或 3 个英文字符"})
		return
	}

	limit := defaultSearchLimit
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	var proj model.Project
	if err := h.db.Where("id = ? AND user_id = ? AND status = 1", projectID, userID).First(&proj).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	like := "%" + escapeLike(query) + "%"
	var docs []model.Document
	// 支持大纲/人物设定/世界观/章节等多种正文类型
	searchTypes := []string{"chapter", "outline", "characters", "worldbuilding"}
	// 去掉自定义 ESCAPE，避免 MySQL "Incorrect arguments to ESCAPE" 错误
	if err := h.db.Select("id", "title", "search_text", "updated_at", "type").
		Where("project_id = ? AND status = 1 AND type IN (?) AND search_text LIKE ?", projectID, searchTypes, like).
		Order("sort_order, id").
		Limit(maxSearchDocs).
		Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]documentSearchMatch, 0, limit)
	total := 0
	needle := strings.ToLower(query)
	for _, doc := range docs {
		matches := buildSearchMatches(doc, needle, query)
		for _, match := range matches {
			total++
			if len(items) < limit {
				items = append(items, match)
			}
		}
	}

	// 同一章节内按 occurrenceIndex 排，章节间按 DB 排序已经稳定；这里仅保护性排序。
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DocID == items[j].DocID {
			return items[i].OccurrenceIndex < items[j].OccurrenceIndex
		}
		return false
	})

	c.JSON(http.StatusOK, gin.H{"data": documentSearchResponse{
		Items:   items,
		Total:   total,
		Limit:   limit,
		HasMore: total > len(items),
	}})
}

func validSearchQuery(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	lettersOrDigits := make([]rune, 0, len(query))
	hasNonASCII := false
	for _, r := range query {
		if unicode.IsSpace(r) {
			continue
		}
		if r > unicode.MaxASCII {
			hasNonASCII = true
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r > unicode.MaxASCII {
			lettersOrDigits = append(lettersOrDigits, r)
		}
	}
	if len(lettersOrDigits) == 0 {
		return false
	}
	if hasNonASCII {
		return len(lettersOrDigits) >= 2
	}
	return len(lettersOrDigits) >= 3
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return replacer.Replace(s)
}

func buildSearchMatches(doc model.Document, needleLower string, originalQuery string) []documentSearchMatch {
	content := doc.SearchText
	if content == "" || needleLower == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	matches := make([]documentSearchMatch, 0)
	occurrence := 0
	for lineIdx, line := range lines {
		lineLower := strings.ToLower(line)
		from := 0
		for {
			idx := strings.Index(lineLower[from:], needleLower)
			if idx == -1 {
				break
			}
			byteIdx := from + idx
			matches = append(matches, documentSearchMatch{
				DocID:           doc.ID,
				Title:           doc.Title,
				Line:            lineIdx + 1,
				Col:             runeColumn(line, byteIdx),
				Snippet:         makeSnippet(line, byteIdx, len([]rune(originalQuery))),
				OccurrenceIndex: occurrence,
			})
			occurrence++
			step := len(needleLower)
			if step <= 0 {
				step = 1
			}
			from = byteIdx + step
			if from >= len(lineLower) {
				break
			}
		}
	}
	return matches
}

func runeColumn(line string, byteIdx int) int {
	if byteIdx <= 0 {
		return 1
	}
	if byteIdx > len(line) {
		byteIdx = len(line)
	}
	return len([]rune(line[:byteIdx])) + 1
}

func makeSnippet(line string, byteIdx int, queryRunes int) string {
	runes := []rune(line)
	col := runeColumn(line, byteIdx) - 1
	if col < 0 {
		col = 0
	}
	if queryRunes <= 0 {
		queryRunes = 1
	}
	// 用户要求：搜索词前后各显示 5 个字
	start := col - 5
	if start < 0 {
		start = 0
	}
	end := col + queryRunes + 5
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		start = end
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet += "…"
	}
	return snippet
}
