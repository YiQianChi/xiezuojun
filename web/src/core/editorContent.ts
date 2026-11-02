// 共享的正文装载工具：主编辑器（Editor.vue）与分屏编辑器（SplitEditor.vue）同源使用，
// 从 Editor.vue 抽出——两处各抄一份会在「标题头剥离/md 表格解析」上静默漂移。
import { stripChapterPrefix } from './chapterTitle'

// ponytail: parse Markdown table blocks (| col | col |) into ProseMirror table nodes
export function parseMarkdownTable(lines: string[], startIdx: number): { tableNode: object; consumed: number } | null {
  const headerLine = lines[startIdx]
  // need at least header + separator + 1 row
  if (startIdx + 2 >= lines.length) return null
  const separatorLine = lines[startIdx + 1]
  // separator must be like |---|---| or | --- | :---: |
  if (!/^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)+\|?\s*$/.test(separatorLine)) return null

  // ponytail: simpler split — strip leading/trailing | then split
  const splitCells = (line: string) => {
    let s = line.trim()
    if (s.startsWith('|')) s = s.slice(1)
    if (s.endsWith('|')) s = s.slice(0, -1)
    return s.split('|').map(c => c.trim())
  }

  const headers = splitCells(headerLine)
  if (headers.length < 1) return null

  // ponytail: parse **bold** and *italic* inline marks into ProseMirror text nodes
  const makeTextContent = (text: string) => {
    if (!text) return []
    const nodes: object[] = []
    // split on **bold** or *italic* patterns
    const regex = /(\*\*[^*]+\*\*|\*[^*]+\*)/g
    let last = 0
    let m: RegExpExecArray | null
    while ((m = regex.exec(text)) !== null) {
      if (m.index > last) nodes.push({ type: 'text', text: text.slice(last, m.index) })
      const matched = m[0]
      if (matched.startsWith('**')) {
        nodes.push({ type: 'text', text: matched.slice(2, -2), marks: [{ type: 'bold' }] })
      } else {
        nodes.push({ type: 'text', text: matched.slice(1, -1), marks: [{ type: 'italic' }] })
      }
      last = m.index + matched.length
    }
    if (last < text.length) nodes.push({ type: 'text', text: text.slice(last) })
    return nodes
  }

  const rows: object[] = []
  // header row
  rows.push({
    type: 'tableRow',
    content: headers.map(h => ({ type: 'tableHeader', content: [{ type: 'paragraph', content: makeTextContent(h) }] }))
  })

  // body rows
  let consumed = 2 // header + separator
  for (let i = startIdx + 2; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('|')) break
    const cells = splitCells(line)
    // pad/truncate to header length
    while (cells.length < headers.length) cells.push('')
    rows.push({
      type: 'tableRow',
      content: cells.slice(0, headers.length).map((c: string) => ({ type: 'tableCell', content: [{ type: 'paragraph', content: makeTextContent(c) }] }))
    })
    consumed++
  }

  if (rows.length < 2) return null // need at least header + 1 body row

  return { tableNode: { type: 'table', content: rows }, consumed }
}

export function editorContentFromText(content: string) {
  if (!content) return { type: 'doc', content: [{ type: 'paragraph' }] }
  const lines = content.split('\n')
  const blocks: object[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    // detect markdown table: line has | and next line is separator
    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?\s*:?-{2,}:?\s*(\|)/.test(lines[i + 1])) {
      const result = parseMarkdownTable(lines, i)
      if (result) {
        blocks.push(result.tableNode)
        i += result.consumed
        continue
      }
    }
    // markdown 标题行：# → h1、## → h2、### → h3（角色卡骨架/导入文本里的标题都能正确渲染）
    const hm = /^(#{1,3})\s+(.*)$/.exec(line)
    if (hm) {
      blocks.push({
        type: 'heading',
        attrs: { level: hm[1].length },
        content: hm[2] ? [{ type: 'text', text: hm[2] }] : [],
      })
      i++
      continue
    }
    blocks.push({
      type: 'paragraph',
      content: line ? [{ type: 'text', text: line }] : []
    })
    i++
  }
  return { type: 'doc', content: blocks }
}

const CONTENT_CHAPTER_PREFIX_RE = /^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回]\s*/

export function stripContentChapterHeading(content: string, title: string): string {
  if (!content) return content
  const bareTitle = stripChapterPrefix(String(title || ''))
  const trimTitle = String(title || '').trim()
  const drop = (text: string): string | null => {
    const normalized = text.replace(CONTENT_CHAPTER_PREFIX_RE, '').trim()
    if (normalized === text && text !== bareTitle && text !== trimTitle) return null // 无号且不是标题 → 作者自己的文字
    if (!normalized || normalized === bareTitle || normalized === trimTitle) return '' // 标题头 → 删
    return null // 带号但内容≠标题（作者自拟章名头）→ 保留
  }
  const s = content.replace(/^\s+/, '')
  if (s.startsWith('<h1')) {
    const openEnd = s.indexOf('>')
    const closeIdx = s.indexOf('</h1>')
    if (openEnd < 0 || closeIdx < 0) return content
    const inner = s.slice(openEnd + 1, closeIdx).replace(/<[^>]+>/g, '').trim()
    if (drop(inner) === '') return s.slice(closeIdx + 5)
    return content
  }
  const lines = content.split('\n')
  const md = /^#\s+(.*)$/.exec(lines[0].trim())
  if (!md) return content
  if (drop(md[1].trim()) === '') {
    lines.shift()
    return lines.join('\n')
  }
  return content
}
