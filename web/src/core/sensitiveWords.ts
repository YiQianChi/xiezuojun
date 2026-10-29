// 敏感词/风险词检测 —— 网文平台（起点/番茄等）审核常见雷区词
// 词库参考行业通行清单（约 74 词），覆盖：涉黄/涉暴/涉政/涉恐/价值观/题材红线/亲属关系（言情审核高发）
// 纯前端扫描，零 AI 成本；命中 ≠ 一定违规，定位给作者发布前自查。

export interface SensitiveHit {
  word: string
  count: number
}

// 中文词直接子串匹配；含 ASCII 字母的词（NP/BDSM）用词边界匹配，防误伤普通英文（如 "NP问题"→"NP问题" 中 NP 独立成词才命中）
const WORDS = [
  // 涉黄
  '性交', '做爱', '性爱', '性器官', '强奸', '轮奸', '迷奸', '春药', '催情', '淫秽',
  '恋童', '幼女', '幼男', '未成年', '童养媳',
  // 涉暴/涉恐
  '肢解', '虐杀', '剥皮', '碎尸', '凌迟', '屠杀', '爆炸物', '买凶', '自残', '自杀',
  '恐怖主义', '贩卖人口', '拐卖', '人贩子',
  // 涉毒
  '贩毒', '吸毒', '制毒',
  // 涉政/价值观
  '英烈', '革命领袖', '国家领导人', '民族歧视', '国家机密', '邪教', '献祭', '活人祭',
  '民族仇恨', '宗教仇恨',
  // 赌博/金融
  '赌博', '赌场', '洗钱', '赌资', '高利贷', '诈骗',
  // 亲属/伦理（言情题材审核高发）
  '乱伦', '伪骨科', '亲兄妹', '亲姐弟', '养父', '养母', '继父', '继母', '小妈',
  '表兄', '表妹', '师生恋', '人兽', '监禁', '高干', '军婚',
  // ASCII 词（词边界匹配）
  'NP', 'BDSM', '训诫',
  // 版权敏感（同人/二创雷区）
  '死亡笔记', '进击的巨人', '东京食尸鬼', '暗杀教室', '寄生兽',
]

const ASCII_WORDS = WORDS.filter(w => /[a-z]/i.test(w))
const CJK_WORDS = WORDS.filter(w => !/[a-z]/i.test(w))

export function scanSensitive(text: string): SensitiveHit[] {
  if (!text) return []
  const hits: SensitiveHit[] = []
  for (const w of CJK_WORDS) {
    const c = countOccurrences(text, w)
    if (c > 0) hits.push({ word: w, count: c })
  }
  for (const w of ASCII_WORDS) {
    const re = new RegExp(`(?<![a-zA-Z])${w}(?![a-zA-Z])`, 'gi')
    const c = (text.match(re) || []).length
    if (c > 0) hits.push({ word: w, count: c })
  }
  hits.sort((a, b) => b.count - a.count)
  return hits
}

function countOccurrences(haystack: string, needle: string): number {
  if (!needle) return 0
  let count = 0
  let i = haystack.indexOf(needle)
  while (i !== -1) {
    count++
    i = haystack.indexOf(needle, i + needle.length)
  }
  return count
}
