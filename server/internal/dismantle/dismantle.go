// Package dismantle 拆书台：把别人的书拆成五件套参考（大纲/人物卡/关系网/节奏/套路）+ 章节标注 + 划词素材。
// 红线：拆书产出只做参考，永不写入用户创作域（记忆/故事线/设定/灵感都不碰）。
// 存储全走 MySQL（sticky 同款模式）；原文分章由前端复用导入建书的解析器完成，后端只收结构化章节。
package dismantle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
)

type Book struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index:idx_dismantle_user_project;size:36"`
	ProjectID uint      `json:"project_id" gorm:"index:idx_dismantle_user_project"`
	Title     string    `json:"title" gorm:"size:120"`
	Words     int       `json:"words"`
	Chapters  int       `json:"chapters"`
	Status    string    `json:"status" gorm:"size:16;default:extracting"` // extracting/done/failed
	Progress  int       `json:"progress"`                                 // 0-100
	Error     string    `json:"error" gorm:"size:500"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Chapter struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	BookID  uint   `json:"book_id" gorm:"index:idx_dismantle_ch_book"`
	Idx     int    `json:"idx"`
	Title   string `json:"title" gorm:"size:150"`
	Content string `json:"content" gorm:"type:mediumtext"`
}

// Output 五件套之一，Payload 为 markdown 纯文本（报告查看器直接渲染）
type Output struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	BookID  uint   `json:"book_id" gorm:"index:idx_dismantle_out_book;uniqueIndex:idx_dismantle_out_book_kind,priority:1"`
	Kind    string `json:"kind" gorm:"size:16;uniqueIndex:idx_dismantle_out_book_kind,priority:2"` // outline/chars/net/pulse/trick
	Payload string `json:"payload" gorm:"type:mediumtext"`
}

type Annotation struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	BookID     uint   `json:"book_id" gorm:"index:idx_dismantle_anno_book"`
	ChapterIdx int    `json:"chapter_idx"`
	Kind       string `json:"kind" gorm:"size:12"` // 钩子/手法/意象/节奏
	Title      string `json:"title" gorm:"size:80"`
	Note       string `json:"note" gorm:"size:600"`
}

// Clipping 划词收藏的拆书素材（独立域，不进灵感库）
type Clipping struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"user_id" gorm:"index:idx_dismantle_clip_user;size:36"`
	ProjectID  uint      `json:"project_id" gorm:"index:idx_dismantle_clip_user"`
	BookID     uint      `json:"book_id"`
	BookTitle  string    `json:"book_title" gorm:"size:120"`
	ChapterIdx int       `json:"chapter_idx"`
	Text       string    `json:"text" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	db       *gorm.DB
	aiClient *ai.Client
)

func SetDB(d *gorm.DB)     { db = d }
func SetAI(c *ai.Client)   { aiClient = c }

const (
	maxBooksPerProject   = 5
	maxChapters          = 3000
	maxTotalBytes        = 30 << 20
	chapterPromptRunes   = 6000 // 单章送 AI 截断
	aggregatePromptRunes = 12000
)

func ownProject(uid string, pid uint) bool {
	var cnt int64
	db.Table("projects").Where("id = ? AND user_id = ? AND status = 1", pid, uid).Count(&cnt)
	return cnt > 0
}

func ownBook(uid string, bookID uint) *Book {
	var b Book
	if err := db.Where("id = ? AND user_id = ?", bookID, uid).First(&b).Error; err != nil {
		return nil
	}
	return &b
}

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/dismantle", handleCreate)
	rg.GET("/dismantle", handleList)
	rg.GET("/dismantle/clips", handleListClips)
	rg.DELETE("/dismantle/clips/:id", handleDeleteClip)
	rg.GET("/dismantle/:id", handleGet)
	rg.GET("/dismantle/:id/chapters/:idx", handleGetChapter)
	rg.POST("/dismantle/:id/clips", handleAddClip)
	rg.DELETE("/dismantle/:id", handleDelete)
}

type createChapterReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// handleCreate POST /dismantle {project_id, title, chapters:[{title,content}]}
// 前端已分好章；后端落库后立刻起后台拆解 goroutine，前端轮询 GET /dismantle/:id 看进度。
func handleCreate(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		ProjectID uint              `json:"project_id"`
		Title     string            `json:"title" binding:"required"`
		Chapters  []createChapterReq `json:"chapters" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：需要书名和章节"})
		return
	}
	if req.ProjectID == 0 || !ownProject(uid, req.ProjectID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project not found"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		req.Title = "未命名拆书"
	}
	if len([]rune(req.Title)) > 60 {
		req.Title = string([]rune(req.Title)[:60])
	}
	if len(req.Chapters) == 0 || len(req.Chapters) > maxChapters {
		c.JSON(http.StatusBadRequest, gin.H{"error": "章节数须在 1~3000 之间"})
		return
	}
	var cnt int64
	db.Model(&Book{}).Where("user_id = ? AND project_id = ?", uid, req.ProjectID).Count(&cnt)
	if cnt >= maxBooksPerProject {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每本书项目最多拆 5 本，请先删除旧的"})
		return
	}
	totalBytes := 0
	words := 0
	for _, ch := range req.Chapters {
		totalBytes += len(ch.Content)
		words += len([]rune(ch.Content))
	}
	if totalBytes > maxTotalBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文本超过 30MB"})
		return
	}

	book := Book{UserID: uid, ProjectID: req.ProjectID, Title: req.Title,
		Words: words, Chapters: len(req.Chapters), Status: "extracting"}
	if err := db.Create(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 章节批量插入（book_id 分批 500）
	for start := 0; start < len(req.Chapters); start += 500 {
		end := start + 500
		if end > len(req.Chapters) {
			end = len(req.Chapters)
		}
		batch := make([]Chapter, 0, end-start)
		for i := start; i < end; i++ {
			title := strings.TrimSpace(req.Chapters[i].Title)
			if title == "" {
				title = fmt.Sprintf("第%d节", i+1)
			}
			if len([]rune(title)) > 100 {
				title = string([]rune(title)[:100])
			}
			batch = append(batch, Chapter{BookID: book.ID, Idx: i + 1, Title: title, Content: req.Chapters[i].Content})
		}
		if err := db.CreateInBatches(batch, 100).Error; err != nil {
			db.Model(&book).Updates(map[string]interface{}{"status": "failed", "error": "章节存储失败"})
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	go runExtract(&book)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": book.ID, "chapters": book.Chapters, "words": book.Words}})
}

func handleList(c *gin.Context) {
	uid := c.GetString("userID")
	pid, _ := strconv.ParseUint(c.Query("project_id"), 10, 64)
	var books []Book
	q := db.Where("user_id = ?", uid)
	if pid > 0 {
		q = q.Where("project_id = ?", uint(pid))
	}
	q.Order("created_at DESC").Find(&books)
	if books == nil {
		books = []Book{}
	}
	c.JSON(http.StatusOK, gin.H{"data": books})
}

// handleGet GET /dismantle/:id — 书元信息 + 五件套 + 章节目录（无正文） + 标注
func handleGet(c *gin.Context) {
	uid := c.GetString("userID")
	book := ownBook(uid, parseUint(c.Param("id")))
	if book == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var outs []Output
	db.Where("book_id = ?", book.ID).Find(&outs)
	kindPayload := map[string]string{}
	for _, o := range outs {
		kindPayload[o.Kind] = o.Payload
	}
	var annos []Annotation
	db.Where("book_id = ?", book.ID).Order("chapter_idx ASC").Find(&annos)
	if annos == nil {
		annos = []Annotation{}
	}
	var chapters []Chapter
	db.Select("id, book_id, idx, title").Where("book_id = ?", book.ID).Order("idx ASC").Find(&chapters)
	toc := make([]gin.H, 0, len(chapters))
	for _, ch := range chapters {
		toc = append(toc, gin.H{"idx": ch.Idx, "title": ch.Title})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"book":        book,
		"outputs":     kindPayload,
		"annotations": annos,
		"toc":         toc,
	}})
}

func handleGetChapter(c *gin.Context) {
	uid := c.GetString("userID")
	book := ownBook(uid, parseUint(c.Param("id")))
	if book == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	idx, _ := strconv.Atoi(c.Param("idx"))
	var ch Chapter
	if err := db.Where("book_id = ? AND idx = ?", book.ID, idx).First(&ch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"idx": ch.Idx, "title": ch.Title, "content": ch.Content}})
}

func handleDelete(c *gin.Context) {
	uid := c.GetString("userID")
	book := ownBook(uid, parseUint(c.Param("id")))
	if book == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	db.Where("book_id = ?", book.ID).Delete(&Chapter{})
	db.Where("book_id = ?", book.ID).Delete(&Output{})
	db.Where("book_id = ?", book.ID).Delete(&Annotation{})
	db.Where("book_id = ? AND user_id = ?", book.ID, uid).Delete(&Clipping{})
	db.Delete(book)
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func handleAddClip(c *gin.Context) {
	uid := c.GetString("userID")
	book := ownBook(uid, parseUint(c.Param("id")))
	if book == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req struct {
		ChapterIdx int    `json:"chapter_idx"`
		Text       string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if len([]rune(req.Text)) > 500 {
		req.Text = string([]rune(req.Text)[:500])
	}
	clip := Clipping{UserID: uid, ProjectID: book.ProjectID, BookID: book.ID, BookTitle: book.Title,
		ChapterIdx: req.ChapterIdx, Text: strings.TrimSpace(req.Text)}
	if err := db.Create(&clip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": clip})
}

func handleListClips(c *gin.Context) {
	uid := c.GetString("userID")
	pid, _ := strconv.ParseUint(c.Query("project_id"), 10, 64)
	var clips []Clipping
	q := db.Where("user_id = ?", uid)
	if pid > 0 {
		q = q.Where("project_id = ?", uint(pid))
	}
	q.Order("created_at DESC").Limit(500).Find(&clips)
	if clips == nil {
		clips = []Clipping{}
	}
	c.JSON(http.StatusOK, gin.H{"data": clips})
}

func handleDeleteClip(c *gin.Context) {
	uid := c.GetString("userID")
	db.Where("id = ? AND user_id = ?", parseUint(c.Param("id")), uid).Delete(&Clipping{})
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func parseUint(s string) uint { v, _ := strconv.ParseUint(s, 10, 64); return uint(v) }

/* ═══════════ 后台拆解 ═══════════ */

func dismantleModel() string {
	if m := getenv("DISMANTLE_MODEL"); m != "" {
		return m
	}
	if m := getenv("MEMORY_MODEL"); m != "" {
		return m
	}
	if m := getenv("AI_MODEL"); m != "" {
		return m
	}
	return "MiniMax-M3"
}

func getenv(k string) string { return strings.TrimSpace(os.Getenv(k)) }

func complete(prompt string) (string, error) {
	if aiClient == nil {
		return "", fmt.Errorf("ai client not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	return aiClient.Complete(ctx, dismantleModel(), []ai.Message{{Role: "user", Content: prompt}})
}

var jsonFenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

type chapterExtract struct {
	Summary string `json:"summary"`
	Anno    []struct {
		Kind  string `json:"kind"`
		Title string `json:"title"`
		Note  string `json:"note"`
	} `json:"anno"`
}

// runExtract 逐章摘要+标注 → 汇总五件套。进度按章推进，失败标记 status=failed（可删可重拆）。
func runExtract(book *Book) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[dismantle] extract panicked book=%d: %v", book.ID, r)
			db.Model(&Book{}).Where("id = ?", book.ID).Updates(map[string]interface{}{"status": "failed", "error": "内部错误"})
		}
	}()
	var chapters []Chapter
	db.Select("idx, title, content").Where("book_id = ?", book.ID).Order("idx ASC").Find(&chapters)
	if len(chapters) == 0 {
		db.Model(&Book{}).Where("id = ?", book.ID).Updates(map[string]interface{}{"status": "failed", "error": "无章节"})
		return
	}

	type chSum struct{ idx int; title, summary string }
	sums := make([]chSum, 0, len(chapters))
	annoRows := make([]Annotation, 0)

	for i, ch := range chapters {
		content := ch.Content
		if len([]rune(content)) > chapterPromptRunes {
			content = string([]rune(content)[:chapterPromptRunes])
		}
		prompt := fmt.Sprintf(`你是资深网文拆书编辑。对下面的小说章节做拆解。
只输出一个 JSON 对象（不要 markdown 代码块、不要解释），格式：
{"summary":"100字以内的剧情摘要","anno":[{"kind":"钩子|手法|意象|节奏","title":"8字内小标题","note":"60字内说明，写可复用的写作技法"}]}
anno 数组 0~3 条，只挑真正值得学的，没有就给空数组。

章节标题：%s

%s`, ch.Title, content)
		raw, err := complete(prompt)
		if err != nil {
			log.Printf("[dismantle] chapter %d failed book=%d: %v", ch.Idx, book.ID, err)
			// 单章失败不终止：降级为截断摘要
			sums = append(sums, chSum{ch.Idx, ch.Title, truncateRunes(ch.Content, 80)})
			db.Model(&Book{}).Where("id = ?", book.ID).Update("progress", (i+1)*100/len(chapters))
			continue
		}
		var ex chapterExtract
		body := strings.TrimSpace(raw)
		if m := jsonFenceRe.FindStringSubmatch(body); m != nil {
			body = strings.TrimSpace(m[1])
		}
		if s := strings.Index(body, "{"); s >= 0 {
			if e := strings.LastIndex(body, "}"); e > s {
				body = body[s : e+1]
			}
		}
		if err := json.Unmarshal([]byte(body), &ex); err != nil || strings.TrimSpace(ex.Summary) == "" {
			sums = append(sums, chSum{ch.Idx, ch.Title, truncateRunes(raw, 100)})
		} else {
			sums = append(sums, chSum{ch.Idx, ch.Title, ex.Summary})
			for _, a := range ex.Anno {
				kind := a.Kind
				switch kind {
				case "钩子", "手法", "意象", "节奏":
				default:
					kind = "手法"
				}
				annoRows = append(annoRows, Annotation{BookID: book.ID, ChapterIdx: ch.Idx,
					Kind: kind, Title: truncateRunes(a.Title, 40), Note: truncateRunes(a.Note, 300)})
			}
		}
		db.Model(&Book{}).Where("id = ?", book.ID).Update("progress", (i+1)*100/len(chapters))
	}

	// 章节摘要拼接文本（五件套的原料）
	var sb strings.Builder
	for _, s := range sums {
		fmt.Fprintf(&sb, "第%d章 %s：%s\n", s.idx, s.title, s.summary)
	}
	digest := truncateRunes(sb.String(), aggregatePromptRunes)
	total := strconv.Itoa(len(chapters))

	aggregates := []struct{ kind, title, promptFmt string }{
		{"outline", "章节大纲", "把这本书按剧情阶段划分（每阶段覆盖若干章），输出 markdown 大纲：每个阶段一行「第X~Y章 阶段名：一句话概括（节奏特征）」。不超过 25 行。\n\n全书共 %s 章，逐章摘要如下：\n\n%s"},
		{"chars", "人物卡", "从逐章摘要中提炼全书主要人物（≤8人），输出 markdown 人物卡，每人一段：**人名**：身份/性格/标志性行为或口头禅/在故事里的功能。只依据摘要，不要编造。\n\n逐章摘要：\n\n%s"},
		{"net", "关系网", "从逐章摘要中提炼人物关系（≤8组），输出 markdown 列表，每组一行：「A — B：关系性质 + 在剧情中的作用」。只依据摘要。\n\n逐章摘要：\n\n%s"},
		{"pulse", "节奏曲线", "分析全书节奏，输出 markdown：①整体节奏结构（开局/中段/高潮的章范围与特征）②压抑-释放循环规律（如有）③钩子密度最高的区段及原因。不超过 15 行。\n\n全书共 %s 章，逐章摘要：\n\n%s"},
		{"trick", "可复用套路", "提炼这本书值得网文作者复用的写作套路（≤8条），输出 markdown 列表，每条：「**套路名**：一句话说明 + 适用场景」。要具体可操作，不要空话。\n\n逐章摘要：\n\n%s"},
	}
	for _, ag := range aggregates {
		var prompt string
		if strings.Contains(ag.promptFmt, "%s") && strings.Count(ag.promptFmt, "%s") == 2 {
			prompt = fmt.Sprintf(ag.promptFmt, total, digest)
		} else {
			prompt = fmt.Sprintf(ag.promptFmt, digest)
		}
		out, err := complete(prompt)
		if err != nil {
			log.Printf("[dismantle] aggregate %s failed book=%d: %v", ag.kind, book.ID, err)
			out = "（生成失败，可删除后重拆）"
		}
		db.Where("book_id = ? AND kind = ?", book.ID, ag.kind).Delete(&Output{})
		db.Create(&Output{BookID: book.ID, Kind: ag.kind, Payload: strings.TrimSpace(out)})
	}

	if len(annoRows) > 0 {
		db.CreateInBatches(annoRows, 100)
	}
	db.Model(&Book{}).Where("id = ?", book.ID).Updates(map[string]interface{}{"status": "done", "progress": 100})
	log.Printf("[dismantle] book=%d done: %d chapters, %d annotations", book.ID, len(chapters), len(annoRows))
}

func truncateRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
