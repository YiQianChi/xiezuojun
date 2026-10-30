package project

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
)

// exportChapterTitle 导出章节名带序号（与前端 chapterDisplayTitle 口径一致）：
// chapter 按全书摊平顺序连续编「第N章 标题」（第一卷 1-4、第二卷续 5-8，Boss 2026-10-30 终版）；
// 标题已带 第X章 前缀或属 序章/楔子/番外/尾声/终章/引子 则原样保留。
var exportSpecialTitleRe = regexp.MustCompile(`^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回]|^(序章|楔子|番外|尾声|终章|引子)`)

// 只用于「不占章节号」判定（序章/楔子/番外类）；标题自带「第X章」前缀的仍占号——与前端摊平口径一致
var exportNoCountTitleRe = regexp.MustCompile(`^(序章|序|楔子|引子|尾声|终章|番外)`)

func exportChapterTitle(ordinal int, title string) string {
	t := strings.TrimSpace(title)
	if t == "" || exportSpecialTitleRe.MatchString(t) {
		return t
	}
	return fmt.Sprintf("第%s章 %s", numToChinese(ordinal), t)
}

// stripChapterHeadingForExport 正文开头若带着老模板写死进正文的「第N章 标题」标题头（HTML h1 或 md # 行），
// 导出时删掉，防止与导出标题行重复编号。安全闸：剥掉章号前缀后的剩余文字必须等于章节标题（或为空）才动，
// 防误伤以「第X章」开头的正文段落。正文源数据不改，只影响导出产物。
var chapterPrefixRe = regexp.MustCompile(`^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回]\s*`)
func stripChapterHeadingForExport(content, title string) string {
	if content == "" {
		return content
	}
	s := strings.TrimLeft(content, " 	\r\n")
	bareTitle := strings.TrimSpace(chapterPrefixRe.ReplaceAllString(strings.TrimSpace(title), ""))
	trimTitle := strings.TrimSpace(title)
	stripPrefix := func(t string) string {
		return strings.TrimSpace(chapterPrefixRe.ReplaceAllString(t, ""))
	}
	if strings.HasPrefix(s, "<h1") {
		openEnd := strings.Index(s, ">")
		closeIdx := strings.Index(s, "</h1>")
		if openEnd < 0 || closeIdx < 0 {
			return content
		}
		inner := strings.TrimSpace(htmlTagRe.ReplaceAllString(s[openEnd+1:closeIdx], ""))
		normalized := stripPrefix(inner)
		if normalized == inner && inner != bareTitle && inner != trimTitle {
			return content // 无章号前缀且不是标题文字 → 正文内容，不动
		}
		if normalized == "" || normalized == bareTitle || normalized == trimTitle {
			return s[closeIdx+5:] // 「第N章 标题」/「第N章」/「标题」标题头 → 删掉（导出已有标题行）
		}
		return content // h1 写的是别的文字，尊重原文
	}
	// markdown 标题行：# 第N章 标题 / # 标题
	lines := strings.Split(content, "\n")
	first := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(first, "#") {
		return content
	}
	first = strings.TrimSpace(strings.TrimLeft(first, "#"))
	normalized := stripPrefix(first)
	if normalized == first && first != bareTitle && first != trimTitle {
		return content
	}
	if normalized == "" || normalized == bareTitle || normalized == trimTitle {
		lines = lines[1:]
		return strings.Join(lines, "\n")
	}
	return content
}

// ExportEPUB 导出 EPUB 电子书
// EPUB 3 最小实现：mimetype + container.xml + content.opf + nav.xhtml + 每章一个 XHTML
// 纯标准库 archive/zip，无外部依赖
func (h *Handler) ExportEPUB(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}

	// 复用 ExportProject 的数据收集逻辑
	docs, proj, ok := h.collectExportDocs(c, uint(id))
	if !ok {
		return
	}

	// 只导出正文类型（chapter/volume）
	var chapters []exportChapter
	chOrdinal := 0
	for _, dc := range docs {
		if dc["type"] == "volume" {
			chapters = append(chapters, exportChapter{Title: getString(dc, "title"), Content: getString(dc, "content")})
		} else if dc["type"] == "chapter" {
			// 特判章节（楔子/番外…）不占号位，与前端卷内口径一致
			if !exportNoCountTitleRe.MatchString(strings.TrimSpace(getString(dc, "title"))) {
				chOrdinal++
			}
			chapters = append(chapters, exportChapter{
				Title:   exportChapterTitle(chOrdinal, getString(dc, "title")),
				Content: stripChapterHeadingForExport(getString(dc, "content"), getString(dc, "title")),
			})
		}
	}
	if len(chapters) == 0 {
		chapters = append(chapters, exportChapter{Title: proj.Name, Content: "（空）"})
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// 1. mimetype（必须第一个，不压缩）
	mimeWriter, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "epub create failed"})
		return
	}
	mimeWriter.Write([]byte("application/epub+zip"))

	// 2. META-INF/container.xml
	writeFile(w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	// 3. OEBPS/content.opf
	var manifestBuf strings.Builder
	manifestBuf.WriteString(`<meta name="cover" content="cover-image"/>` + "\n")
	var spineBuf strings.Builder
	for i := range chapters {
		filename := fmt.Sprintf("chapter-%04d.xhtml", i+1)
		manifestBuf.WriteString(fmt.Sprintf(`<item id="chap%d" href="%s" media-type="application/xhtml+xml"/>`+"\n", i+1, filename))
		spineBuf.WriteString(fmt.Sprintf(`<itemref idref="chap%d"/>`+"\n", i+1))
	}
	opf := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="bookid">xiezuojun-%d</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:language>zh-CN</dc:language>
    <dc:creator>%s</dc:creator>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
%s  </manifest>
  <spine>
%s  </spine>
</package>`,
		id, escapeXML(proj.Name), "写作菌用户", time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		manifestBuf.String(), spineBuf.String())
	writeFile(w, "OEBPS/content.opf", opf)

	// 4. OEBPS/nav.xhtml（目录）
	var navBuf strings.Builder
	navBuf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><meta charset="utf-8"/><title>目录</title></head>
<body>
<nav epub:type="toc">
<ol>`)
	for i, ch := range chapters {
		navBuf.WriteString(fmt.Sprintf(`<li><a href="chapter-%04d.xhtml">%s</a></li>`, i+1, escapeXML(ch.Title)))
	}
	navBuf.WriteString(`</ol>
</nav>
</body>
</html>`)
	writeFile(w, "OEBPS/nav.xhtml", navBuf.String())

	// 5. 每章一个 XHTML 文件
	for i, ch := range chapters {
		filename := fmt.Sprintf("OEBPS/chapter-%04d.xhtml", i+1)
		body := markdownToXHTML(ch.Content)
		xhtml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><meta charset="utf-8"/><title>%s</title></head>
<body>
<h1>%s</h1>
%s
</body>
</html>`, escapeXML(ch.Title), escapeXML(ch.Title), body)
		writeFile(w, filename, xhtml)
	}

	if err := w.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "epub close failed"})
		return
	}

	c.Header("Content-Type", "application/epub+zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.epub"`, sanitizeFilename(proj.Name)))
	c.Data(http.StatusOK, "application/epub+zip", buf.Bytes())
}

// ExportDOCX 导出 Word 文档（最小 DOCX 实现，纯标准库）
// DOCX = ZIP 包含 [Content_Types].xml + _rels/.rels + word/document.xml
func (h *Handler) ExportDOCX(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	docs, proj, ok := h.collectExportDocs(c, uint(id))
	if !ok {
		return
	}

	var chapters []exportChapter
	chOrdinal := 0
	for _, dc := range docs {
		if dc["type"] == "volume" {
			chapters = append(chapters, exportChapter{Title: getString(dc, "title"), Content: getString(dc, "content")})
		} else if dc["type"] == "chapter" {
			// 特判章节（楔子/番外…）不占号位，与前端卷内口径一致
			if !exportNoCountTitleRe.MatchString(strings.TrimSpace(getString(dc, "title"))) {
				chOrdinal++
			}
			chapters = append(chapters, exportChapter{
				Title:   exportChapterTitle(chOrdinal, getString(dc, "title")),
				Content: stripChapterHeadingForExport(getString(dc, "content"), getString(dc, "title")),
			})
		}
	}
	if len(chapters) == 0 {
		chapters = append(chapters, exportChapter{Title: proj.Name, Content: "（空）"})
	}

	// 构建 word/document.xml 主体
	var body strings.Builder
	for _, ch := range chapters {
		// 标题样式
		body.WriteString(fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, escapeXML(ch.Title)))
		// 正文段落
		plain := stripMarkdown(ch.Content)
		for _, line := range strings.Split(plain, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			body.WriteString(fmt.Sprintf(`<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, escapeXML(line)))
		}
	}

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>` + body.String() + `</w:body>
</w:document>`

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// [Content_Types].xml
	writeFile(w, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)

	// _rels/.rels
	writeFile(w, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`)

	// word/document.xml
	writeFile(w, "word/document.xml", documentXML)

	if err := w.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "docx close failed"})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.docx"`, sanitizeFilename(proj.Name)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", buf.Bytes())
}

// exportChapter 导出用章节结构
type exportChapter struct {
	Title   string
	Content string
}

// collectExportDocs 复用 ExportProject 数据收集
func (h *Handler) collectExportDocs(c *gin.Context, projectID uint) ([]map[string]interface{}, model.Project, bool) {
	var proj model.Project
	if err := h.db.First(&proj, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return nil, proj, false
	}
	var docs []model.Document
	h.db.Where("project_id = ? AND status = 1", projectID).Order("sort_order, id").Find(&docs)
	// 树深度优先摊平（根级按 sort_order 交错卷/散章，卷内按各自 sort_order）——
	// 卷和章节的 sort_order 是两套父级序列，全表混排顺序会碰运气（Boss 2026-10-30 修）
	ordered := flattenDocTree(docs)
	docContents := make([]map[string]interface{}, 0)
	for _, doc := range ordered {
		content, _ := h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
		if content == "" && doc.ParentID != nil && *doc.ParentID > 0 {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
		} else if content == "" {
			content, _ = h.ds.loadDocContentFallback(doc.ID)
		}
		docContents = append(docContents, map[string]interface{}{
			"id": doc.ID, "title": doc.Title, "type": doc.Type, "content": content,
		})
	}
	return docContents, proj, true
}

// flattenDocTree 文档树深度优先摊平：根级节点按 sort_order/id 排，进入文件夹后子级按各自 sort_order/id
func flattenDocTree(docs []model.Document) []model.Document {
	childrenOf := make(map[int][]model.Document)
	var roots []model.Document
	for _, d := range docs {
		if d.ParentID != nil && *d.ParentID > 0 {
			childrenOf[int(*d.ParentID)] = append(childrenOf[int(*d.ParentID)], d)
		} else {
			roots = append(roots, d)
		}
	}
	bySort := func(list []model.Document) []model.Document {
		out := append([]model.Document(nil), list...)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].SortOrder != out[j].SortOrder {
				return out[i].SortOrder < out[j].SortOrder
			}
			return out[i].ID < out[j].ID
		})
		return out
	}
	var walk func(list []model.Document) []model.Document
	walk = func(list []model.Document) []model.Document {
		result := make([]model.Document, 0, len(list))
		for _, d := range bySort(list) {
			result = append(result, d)
			if kids, ok := childrenOf[int(d.ID)]; ok {
				result = append(result, walk(kids)...)
			}
		}
		return result
	}
	return walk(roots)
}

// ExportTXT 导出纯文本
func (h *Handler) ExportTXT(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	docs, proj, ok := h.collectExportDocs(c, uint(id))
	if !ok {
		return
	}
	var buf strings.Builder
	buf.WriteString(proj.Name + "\n\n")
	txtOrdinal := 0
	for _, dc := range docs {
		if dc["type"] != "chapter" && dc["type"] != "volume" {
			continue
		}
		title := getString(dc, "title")
		if dc["type"] == "chapter" {
			if !exportNoCountTitleRe.MatchString(strings.TrimSpace(title)) {
				txtOrdinal++
			}
			title = exportChapterTitle(txtOrdinal, title)
		}
		buf.WriteString(title + "\n")
		buf.WriteString(strings.Repeat("─", 20) + "\n")
		buf.WriteString(stripMarkdown(stripChapterHeadingForExport(getString(dc, "content"), getString(dc, "title"))) + "\n\n")
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.txt"`, sanitizeFilename(proj.Name)))
	c.String(http.StatusOK, buf.String())
}

// ===== 工具函数 =====

func writeFile(w *zip.Writer, name, content string) {
	f, _ := w.Create(name)
	f.Write([]byte(content))
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// markdownToXHTML 将简单 Markdown 转为 XHTML（段落 + 换行）
// ponytail: 不引入 markdown 解析库，写作菌正文就是纯文本+换行
func markdownToXHTML(md string) string {
	if md == "" {
		return "<p></p>"
	}
	lines := strings.Split(md, "\n")
	var buf strings.Builder
	inPara := false
	flush := func() {
		if inPara {
			buf.WriteString("</p>\n")
			inPara = false
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		// 标题
		if strings.HasPrefix(trimmed, "### ") {
			flush()
			buf.WriteString("<h3>" + escapeXML(trimmed[4:]) + "</h3>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			buf.WriteString("<h2>" + escapeXML(trimmed[3:]) + "</h2>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			flush()
			buf.WriteString("<h1>" + escapeXML(trimmed[2:]) + "</h1>\n")
			continue
		}
		if !inPara {
			buf.WriteString("<p>")
			inPara = true
		} else {
			buf.WriteString("<br/>")
		}
		buf.WriteString(escapeXML(trimmed))
	}
	flush()
	return buf.String()
}

// stripMarkdown 去除 Markdown 标记，输出纯文本
// stripHTMLTags 导出兜底：编辑器自动保存的章节 content 是 HTML（<p>段</p>），
// 导出前把块级闭合标签转换行、其余标签剥掉、解码常见实体——否则 TXT/DOCX/EPUB 里出现字面 <p>。
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var htmlEntity = map[string]string{"&nbsp;": " ", "&amp;": "&", "&lt;": "<", "&gt;": ">", "&quot;": `"`, "&#39;": "'", "&ldquo;": "“", "&rdquo;": "”"}

func stripHTMLTags(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	for _, tag := range []string{"</p>", "</h1>", "</h2>", "</h3>", "</div>", "</blockquote>", "</li>"} {
		s = strings.ReplaceAll(s, tag, "\n")
	}
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	for k, v := range htmlEntity {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func stripMarkdown(md string) string {
	md = stripHTMLTags(md)
	var buf strings.Builder
	for _, line := range strings.Split(md, "\n") {
		l := strings.TrimSpace(line)
		l = strings.TrimPrefix(l, "#")
		l = strings.TrimSpace(l)
		l = strings.TrimPrefix(l, ">")
		l = strings.TrimSpace(l)
		l = strings.TrimPrefix(l, "- ")
		l = strings.TrimPrefix(l, "* ")
		if l != "" {
			buf.WriteString(l + "\n")
		}
	}
	return buf.String()
}
