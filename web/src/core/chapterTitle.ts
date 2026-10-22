// 章节显示标题工具：树/顶栏统一用「第N章 名称」，序号按章节在同级中的位置自动算，
// 作者只需要写章节名（Boss 2026-10-22：不要让作者手动写第几章）

const CHAPTER_PREFIX_RE = /^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回]\s*/
// 序章/楔子/番外这类标题自带位置含义，不参与自动编号
const NO_NUMBER_RE = /^(序章|序|楔子|引子|尾声|终章|番外)/

const CN_DIGITS = '零一二三四五六七八九'

export function numToChinese(n: number): string {
  if (n <= 0) return String(n)
  if (n < 10) return CN_DIGITS[n]
  if (n < 20) return '十' + (n % 10 ? CN_DIGITS[n % 10] : '')
  if (n < 100) return CN_DIGITS[Math.floor(n / 10)] + '十' + (n % 10 ? CN_DIGITS[n % 10] : '')
  if (n < 1000) {
    const h = Math.floor(n / 100)
    const r = n % 100
    return CN_DIGITS[h] + '百' + (r === 0 ? '' : r < 10 ? '零' + CN_DIGITS[r] : numToChinese(r))
  }
  return String(n) // 千章以上直接阿拉伯数字
}

/** 剥掉作者手写的「第X章」前缀，防止和自动编号叠床架屋 */
export function stripChapterPrefix(title: string): string {
  return String(title || '').replace(CHAPTER_PREFIX_RE, '').trim()
}

/** 章节在同级（同卷/同父级）中的序号，从 1 开始；找不到返回 0 */
export function chapterOrdinal(doc: any, allDocs: any[]): number {
  const siblings = allDocs
    .filter((d: any) => d.type === 'chapter' && (d.parent_id || null) === (doc.parent_id || null))
    .sort((a: any, b: any) => (a.sort_order || 0) - (b.sort_order || 0))
  return siblings.findIndex((d: any) => d.id === doc.id) + 1
}

/** 章节显示标题：「第一章 雨夜」；序章/番外类保持原名 */
export function chapterDisplayTitle(doc: any, allDocs: any[]): string {
  const raw = stripChapterPrefix(doc?.title)
  if (NO_NUMBER_RE.test(raw)) return raw
  const ord = chapterOrdinal(doc, allDocs)
  if (ord <= 0) return doc?.title || ''
  return raw ? `第${numToChinese(ord)}章 ${raw}` : `第${numToChinese(ord)}章`
}
