package project

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
)

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
	for _, dc := range docs {
		if dc["type"] == "chapter" || dc["type"] == "volume" {
			chapters = append(chapters, exportChapter{
				Title:   getString(dc, "title"),
				Content: getString(dc, "content"),
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
	docContents := make([]map[string]interface{}, 0)
	for _, doc := range docs {
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
	for _, dc := range docs {
		if dc["type"] != "chapter" && dc["type"] != "volume" {
			continue
		}
		buf.WriteString(getString(dc, "title") + "\n")
		buf.WriteString(strings.Repeat("─", 20) + "\n")
		buf.WriteString(stripMarkdown(getString(dc, "content")) + "\n\n")
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
func stripMarkdown(md string) string {
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
