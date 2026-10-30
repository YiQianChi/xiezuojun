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

/** 章节序号 = 同父级（卷内）独立编号（Boss 2026-10-30 分离方案定稿：编号只是显示标签，
 *  正文 H1 永不带号，导出时按卷统一编号——插章只影响本卷，其他卷一个字不动）。序章/番外类不占号位。 */
export function chapterOrdinal(doc: any, allDocs: any[]): number {
  const siblings = allDocs
    .filter((d: any) => d.type === 'chapter'
      && (d.parent_id || null) === (doc.parent_id || null)
      && !NO_NUMBER_RE.test(String(d.title || '').trim()))
    .sort((a: any, b: any) => (a.sort_order || 0) - (b.sort_order || 0))
  return siblings.findIndex((d: any) => d.id === doc.id) + 1
}

/** 章节显示标题：「第一章 雨夜」；序章/番外类保持原名 */
export function chapterDisplayTitle(doc: any, allDocs: any[]): string {
  const rawTitle = String(doc?.title || '').trim()
  if (NO_NUMBER_RE.test(rawTitle)) return rawTitle
  // 作者/导入原稿自带的「第X章」编号优先，原样显示（导入保真 + 作者明确指定）
  if (CHAPTER_PREFIX_RE.test(rawTitle)) return rawTitle
  const ord = chapterOrdinal(doc, allDocs)
  if (ord <= 0) return doc?.title || ''
  return rawTitle ? `第${numToChinese(ord)}章 ${rawTitle}` : `第${numToChinese(ord)}章`
}
