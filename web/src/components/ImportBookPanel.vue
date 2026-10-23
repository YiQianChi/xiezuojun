<template>
  <div class="import-panel">
    <!-- 第一步：选文件 -->
    <template v-if="step === 'pick'">
      <div
        class="drop-zone"
        :class="{ dragover }"
        @click="fileInput?.click()"
        @dragover.prevent="dragover = true"
        @dragleave="dragover = false"
        @drop.prevent="onDrop"
      >
        <div class="drop-icon">📄</div>
        <p>点击选择或拖入 .txt 文件</p>
        <p class="drop-hint">自动识别 UTF-8 / GBK 编码，按「第X章」分章，全程不经过 AI</p>
      </div>
      <input ref="fileInput" type="file" accept=".txt,text/plain" hidden @change="onPick" />
    </template>

    <!-- 第三步：导入进度 -->
    <template v-else-if="step === 'importing'">
      <div class="import-progress-box">
        <div class="ip-text">{{ importStatus }}</div>
        <div class="ip-bar">
          <div class="ip-fill" :class="{ shimmer: importingActive }" :style="{ width: importPct + '%' }">
            <div class="ip-shine"></div>
          </div>
        </div>
        <div class="ip-count">
          <span class="ip-pct">{{ importPct }}%</span>
          {{ importedChapters }} / {{ totalChaptersToImport }} 章
        </div>
      </div>
    </template>

    <!-- 第二步：预览分章 -->
    <template v-else>
      <div class="form-group">
        <label>书籍名称</label>
        <input v-model="bookName" class="input" placeholder="输入书籍名称" />
      </div>

      <template v-if="volumes.length > 0">
        <div class="stat-row">
          识别到 <strong>{{ volumes.length }}</strong> 卷 / <strong>{{ totalChapters }}</strong> 章，共 <strong>{{ totalChars }}</strong> 字
        </div>
        <!-- 标题格式不一致 → 询问是否统一（默认保留原格式，确认后才统一） -->
        <div v-if="needsNormalize" class="unify-box">
          <div class="unify-title">⚠️ 检测到章节标题格式不一致</div>
          <div v-for="ex in normalizeExamples" :key="ex.raw" class="unify-ex">
            <span class="ex-raw">{{ ex.raw }}</span> → <span class="ex-norm">{{ ex.norm }}</span>
          </div>
          <label class="radio-row">
            <input type="radio" v-model="unifyTitles" :value="true" />
            统一为「第N章 标题」格式
          </label>
          <label class="radio-row">
            <input type="radio" v-model="unifyTitles" :value="false" />
            保留原格式导入
          </label>
        </div>
        <div class="chapter-preview">
          <div v-for="(v, vi) in effectiveVolumes" :key="vi" class="pv-vol">
            <div class="pv-vol-title">📁 {{ v.title || '第一卷' }}</div>
            <div v-for="(ch, ci) in v.chapters" :key="ci" class="pv-ch">
              <span class="pv-ch-title">📄 {{ ch.title }}</span>
              <span class="pv-ch-words">{{ countChars(ch.content) }} 字</span>
            </div>
          </div>
        </div>
      </template>

      <!-- 没识别到章节标题 → 兜底选项 -->
      <template v-else>
        <div class="warn-box">
          没识别到「第X章」这类章节标题。选择怎么处理：
        </div>
        <label class="radio-row">
          <input type="radio" v-model="fallbackMode" value="single" />
          整本作为一章
        </label>
        <label class="radio-row">
          <input type="radio" v-model="fallbackMode" value="split" />
          每
          <input type="number" v-model.number="splitSize" class="input split-input" min="500" max="20000" />
          字切一章（在段落边界下刀）
        </label>
        <div class="stat-row">全文共 <strong>{{ countChars(rawText) }}</strong> 字</div>
      </template>

      <div class="panel-actions">
        <button class="btn" @click="step = 'pick'">重新选择</button>
        <button class="btn primary" @click="doImport" :disabled="importing || !canSubmit">
          {{ importing ? '导入中…' : '开始导入' }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { projectApi } from '../core/api'
import { toast } from '../composables/useToast'

const emit = defineEmits<{
  imported: [projectId: number]
  'update:modelValue': [v: boolean]
}>()

interface ParsedChapter { title: string; content: string }
interface ParsedVolume { title: string; chapters: ParsedChapter[] }

const step = ref<'pick' | 'preview' | 'importing'>('pick')
const dragover = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const bookName = ref('')
const rawText = ref('')
const volumes = ref<ParsedVolume[]>([])
const fallbackMode = ref<'single' | 'split'>('split')
const splitSize = ref(3000)
// 标题格式统一：默认保留原格式，用户确认后才统一
const unifyTitles = ref(false)
const importing = ref(false)
// 分批导入进度
const importStatus = ref('')
const importedChapters = ref(0)
const totalChaptersToImport = ref(0)
// 导入进行中标志（驱动 shimmer 动效，完成/失败时停）
const importingActive = ref(false)
const importPct = computed(() =>
  totalChaptersToImport.value > 0 ? Math.min(100, Math.round(importedChapters.value / totalChaptersToImport.value * 100)) : 0)

// 章节/卷标题行识别：整行匹配且限制长度，防止误吃正文里提到「第三章」的句子
// 五种现实格式：①第X章/节/回（尾巴禁逗号句中句号，仅允许结尾句号——真实章名如「第914章 蔚蔚的心里障碍。」）②行首纯数字序号「752.标题」③「第050.标题」省略「章」字（尾巴放宽到 78，标题常与正文首句粘同一行）④Chapter N ⑤特殊章
// 另：原始行以全角空格/tab 缩进开头 = 正文段，直接跳过标题判定（见 parseVolumes）
const VOLUME_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*卷[^\n]{0,40}|卷\s*[0-9零一二三四五六七八九十百千万两]+[^\n]{0,30})[ \t　]*$/
const CHAPTER_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回话](?:[^，。！？\n]{0,40}[。！？]?)?|第\s*[0-9]{1,4}\s*[、.．]\s*[^\s第][^\n]{0,78}|序章|楔子|引子|尾声|终章|番外[^\n]{0,30}|Chapter\s+\d+[^\n]{0,40}|[0-9]{1,4}[、.．]\s*[^\s第][^\n]{0,38})[ \t　]*$/

const totalChapters = computed(() => volumes.value.reduce((n, v) => n + v.chapters.length, 0))
const totalChars = computed(() =>
  volumes.value.reduce((n, v) => n + v.chapters.reduce((m, c) => m + countChars(c.content), 0), 0))
const canSubmit = computed(() =>
  bookName.value.trim().length > 0 && (volumes.value.length > 0 ? totalChapters.value > 0 : rawText.value.trim().length > 0))

// —— 标题格式一致性检测 ——
// 有一章归一后和原标题不一样 = 全书存在格式不一致，提示用户选择（默认保留原格式）
const needsNormalize = computed(() =>
  volumes.value.some(v => v.chapters.some(c => normalizeChapterTitle(c.title) !== c.title)))

// 不一致时展示 3 个「原样 → 统一后」对照示例（每行去重）
const normalizeExamples = computed(() => {
  const seen = new Set<string>()
  const list: { raw: string; norm: string }[] = []
  for (const v of volumes.value) {
    for (const c of v.chapters) {
      const norm = normalizeChapterTitle(c.title)
      if (norm !== c.title && !seen.has(c.title)) {
        seen.add(c.title)
        list.push({ raw: c.title, norm })
        if (list.length >= 3) return list
      }
    }
  }
  return list
})

// 用户选了统一 → 预览和导入都用归一后的标题；否则用原始标题
const effectiveVolumes = computed(() => volumes.value.map(v => ({
  title: v.title,
  chapters: v.chapters.map(c => ({ ...c, title: unifyTitles.value ? normalizeChapterTitle(c.title) : c.title })),
})))

// 外部（新建弹窗关闭时）调用来重置面板
function reset() {
  step.value = 'pick'
  bookName.value = ''
  rawText.value = ''
  volumes.value = []
  unifyTitles.value = false
  importing.value = false
}
defineExpose({ reset })

function countChars(s: string) {
  return (s || '').replace(/\s/g, '').length
}

// 网文 TXT 两大编码：先试 UTF-8（严格模式，有错必抛），失败回退 GBK
async function readTextFile(file: File): Promise<string> {
  const buf = await file.arrayBuffer()
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buf)
  } catch {
    return new TextDecoder('gbk').decode(buf)
  }
}

async function onPick(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) await loadFile(f)
  ;(e.target as HTMLInputElement).value = ''
}

async function onDrop(e: DragEvent) {
  dragover.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) await loadFile(f)
}

async function loadFile(file: File) {
  if (!/\.txt$/i.test(file.name)) {
    toast.error('目前只支持 .txt 文件')
    return
  }
  if (file.size > 30 * 1024 * 1024) {
    toast.error('文件超过 30MB，请拆分后导入')
    return
  }
  try {
    rawText.value = await readTextFile(file)
  } catch {
    toast.error('文件读取失败')
    return
  }
  bookName.value = file.name.replace(/\.txt$/i, '')
  volumes.value = parseVolumes(rawText.value)
  step.value = 'preview'
}

// 标题归一：把「752.标题」「第752章.标题」等杂式统一成「第752章 标题」，
// 和全书前面的章节格式保持一致（Boss 2026-10-23：752 这样的也都改成统一格式）
function normalizeChapterTitle(raw: string): string {
  const t = raw.trim()
  // 行首纯数字序号：752.标题 / 752、标题 / 752．标题 → 第752章 标题
  const numMatch = t.match(/^([0-9]{1,4})\s*[、.．]\s*(.+)$/)
  if (numMatch) {
    return `第${numMatch[1]}章 ${numMatch[2].trim()}`
  }
  // 「第050.标题」（省略「章」字）→ 第50章 标题
  const diMatch = t.match(/^第\s*([0-9]{1,4})\s*[、.．]\s*(.+)$/)
  if (diMatch) {
    return `第${diMatch[1]}章 ${diMatch[2].trim()}`
  }
  // 「第752章.标题」「第 752 章 标题」→ 第752章 标题（序号与名字之间统一空格）
  const cnMatch = t.match(/^(第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回话])\s*[.．、:：]?\s*(.*)$/)
  if (cnMatch) {
    const name = cnMatch[2].trim()
    // 序号统一用阿拉伯数字（「第七百五十二章」→「第752章」；中文数字转不了就原样保留）
    const numStr = chineseToArabic(cnMatch[1])
    return name ? `第${numStr}章 ${name}` : `第${numStr}章`
  }
  return t
}

// 中文数字序号 → 阿拉伯数字（只处理到万级，章节号够用）
function chineseToArabic(prefix: string): string {
  const m = prefix.match(/^第\s*([0-9]+|[零一二三四五六七八九十百千万两]+)\s*[章节回话]$/)
  if (!m) return prefix.replace(/^第\s*|\s*[章节回话]$/g, '')
  const s = m[1]
  if (/^[0-9]+$/.test(s)) return s
  const digit: Record<string, number> = { 零: 0, 一: 1, 二: 2, 两: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9 }
  let total = 0, section = 0, num = 0
  for (const ch of s) {
    if (digit[ch] !== undefined) { num = digit[ch] }
    else if (ch === '十') { section += (num || 1) * 10; num = 0 }
    else if (ch === '百') { section += (num || 1) * 100; num = 0 }
    else if (ch === '千') { section += (num || 1) * 1000; num = 0 }
    else if (ch === '万') { total += (section + num) * 10000; section = 0; num = 0 }
  }
  const result = total + section + num
  return result > 0 ? String(result) : s
}

function parseVolumes(text: string): ParsedVolume[] {
  interface RawChapter { title: string; body: string[] }
  interface RawVolume { title: string; chapters: RawChapter[] }
  const lines = text.split(/\r\n|\r|\n/)
  const vols: RawVolume[] = []
  let curVol: RawVolume | null = null
  let curCh: RawChapter | null = null
  let foundChapter = false
  const preamble: string[] = []

  const ensureVol = () => {
    if (!curVol) { curVol = { title: '', chapters: [] }; vols.push(curVol) }
    return curVol
  }

  for (const line of lines) {
    // 原始行以全角空格缩进开头 = 正文段（如「　　第一章的正文…」），不当标题——防正文段恰好以「第X章」开头被误切
    if (/^[　\t]{1,}/.test(line)) {
      if (curCh) curCh.body.push(line)
      else preamble.push(line)
      continue
    }
    const t = line.trim()
    if (t && t.length <= 60 && VOLUME_RE.test(t)) {
      curVol = { title: t, chapters: [] }
      vols.push(curVol)
      curCh = null
      continue
    }
    // 「第N.标题+正文粘一行」这类格式可到 90+ 字（如第194章那行 60 字），长度上限交给 CHAPTER_RE 各分支自己管
    if (t && t.length <= 95 && CHAPTER_RE.test(t)) {
      foundChapter = true
      // 存原始标题：是否归一成「第N章 标题」由用户在预览时决定
      curCh = { title: t, body: [] }
      ensureVol().chapters.push(curCh)
      continue
    }
    if (curCh) curCh.body.push(line)
    else preamble.push(line)
  }

  // 一个章节标题都没识别到 → 返回空，交给「整本一章/按字数切」兜底流程
  if (!foundChapter) return []

  // 收尾：body 转字符串、去首尾空行；空卷丢掉
  const result: ParsedVolume[] = vols
    .map(v => ({
      title: v.title,
      chapters: v.chapters.map(c => ({
        title: c.title,
        content: c.body.join('\n').replace(/^\n+|\n+$/g, ''),
      })),
    }))
    .filter(v => v.chapters.length > 0)

  // 第一个章节标题之前的内容不丢——作为「未命名部分」放到最前
  const pre = preamble.join('\n').trim()
  if (pre) {
    if (result.length === 0) result.push({ title: '', chapters: [] })
    result[0].chapters.unshift({ title: '未命名部分（正文开头）', content: pre })
  }
  return result
}

// 无章节标题时的兜底：按字数在段落边界切
function splitBySize(text: string, size: number): ParsedVolume[] {
  const paras = text.split(/\n+/)
  const chapters: ParsedChapter[] = []
  let buf: string[] = []
  let bufLen = 0
  const flush = () => {
    if (!buf.length) return
    chapters.push({ title: `第 ${chapters.length + 1} 部分`, content: buf.join('\n') })
    buf = []; bufLen = 0
  }
  for (const p of paras) {
    buf.push(p)
    bufLen += p.length
    if (bufLen >= size) flush()
  }
  flush()
  return [{ title: '', chapters }]
}

async function doImport() {
  if (!canSubmit.value || importing.value) return
  importing.value = true
  step.value = 'importing'
  importStatus.value = '准备导入…'
  try {
    let vols = effectiveVolumes.value
    if (vols.length === 0) {
      vols = fallbackMode.value === 'single'
        ? [{ title: '', chapters: [{ title: '全文', content: rawText.value.trim() }] }]
        : splitBySize(rawText.value.trim(), Math.max(500, splitSize.value || 3000))
    }

    // 摊平成章流再按 500 章一批切（跨卷的批允许同批带多个卷，后端按卷名归并）
    const chapterStream: { volTitle: string; chapter: ParsedChapter }[] = []
    for (const v of vols) {
      const vt = v.title || ''
      for (const ch of v.chapters) chapterStream.push({ volTitle: vt, chapter: ch })
    }
    totalChaptersToImport.value = chapterStream.length
    importedChapters.value = 0
    importingActive.value = true

    const BATCH = 500
    let projectId: number | null = null
    let firstBatch = true

    for (let i = 0; i < chapterStream.length; i += BATCH) {
      const slice = chapterStream.slice(i, i + BATCH)
      // 聚成卷结构
      const batchVols: { title: string; chapters: ParsedChapter[] }[] = []
      for (const item of slice) {
        const last = batchVols[batchVols.length - 1]
        if (last && last.title === item.volTitle) last.chapters.push(item.chapter)
        else batchVols.push({ title: item.volTitle, chapters: [item.chapter] })
      }

      const batchNo = Math.floor(i / BATCH) + 1
      importStatus.value = firstBatch ? '创建书籍并导入第一批…' : `导入中（第 ${batchNo} 批）…`
      // 请求发出前先推进度到本批中点，让条在请求期间就"在动"
      importedChapters.value = Math.min(i + Math.floor(slice.length / 2), totalChaptersToImport.value)
      const res: any = firstBatch
        ? await projectApi.importBook({ name: bookName.value.trim(), volumes: batchVols })
        : await projectApi.importBookBatch({ project_id: projectId!, name: bookName.value.trim(), volumes: batchVols })
      const data = res.data?.data || res.data
      projectId = data.project_id ?? projectId
      importedChapters.value = Math.min(i + slice.length, totalChaptersToImport.value)
      firstBatch = false
    }

    importingActive.value = false
    importStatus.value = '导入完成'
    toast.success(`导入完成：${importedChapters.value} 章已上架`)
    emit('update:modelValue', false)
    emit('imported', projectId!)
  } catch (e: any) {
    importingActive.value = false
    toast.error(e.response?.data?.error || '导入失败，请重试')
    step.value = 'preview'
  } finally {
    importing.value = false
  }
}
</script>

<style scoped>
.drop-zone {
  border: 2px dashed var(--border, #e0e0e0);
  border-radius: 12px;
  padding: 36px 20px;
  text-align: center;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all .15s;
}

/* 导入进度 */
.import-progress-box { padding: 30px 10px 24px; }
.ip-text { font-size: 14px; color: var(--text-primary); margin-bottom: 14px; }
.ip-bar {
  height: 8px;
  border-radius: 4px;
  background: var(--border, #e5e5ea);
  overflow: hidden;
}
.ip-fill {
  height: 100%;
  background: var(--accent, #007aff);
  border-radius: 4px;
  transition: width .4s ease;
}
.ip-count {
  font-size: 12px;
  color: var(--text-tertiary);
  text-align: center;
  margin-top: 10px;
  font-variant-numeric: tabular-nums;
}
.ip-pct {
  font-size: 22px;
  font-weight: 700;
  color: var(--accent, #007aff);
  margin-right: 10px;
  font-variant-numeric: tabular-nums;
}
/* 请求进行中的流光动效：一道高光从左到右循环滑过 */
.ip-fill { position: relative; overflow: hidden; }
.ip-fill.shimmer .ip-shine {
  position: absolute;
  top: 0; bottom: 0; left: 0;
  width: 40%;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,.45), transparent);
  animation: ip-shine 1.2s ease-in-out infinite;
}
@keyframes ip-shine {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(250%); }
}
.drop-zone:hover, .drop-zone.dragover {
  border-color: var(--accent, #007aff);
  background: rgba(0,122,255,.04);
}
.drop-icon { font-size: 40px; margin-bottom: 8px; }
.drop-hint { font-size: 12px; color: var(--text-tertiary); margin-top: 6px; }
.form-group { margin-bottom: 14px; }
.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}
.input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  outline: none;
  font-size: 14px;
  color: var(--text-primary);
  background: var(--bg-secondary, #fff);
  box-sizing: border-box;
}
.input:focus { border-color: var(--accent, #007aff); }
.stat-row {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 10px;
}
.stat-row strong { color: var(--accent, #007aff); }
.chapter-preview {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid var(--border, #eee);
  border-radius: 10px;
  padding: 8px 12px;
}
.pv-vol-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  padding: 6px 0 2px;
}
.pv-ch {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  color: var(--text-secondary);
  padding: 3px 0 3px 16px;
}
.pv-ch-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pv-ch-words { flex-shrink: 0; color: var(--text-tertiary); }
.warn-box {
  font-size: 13px;
  color: #b45309;
  background: rgba(245,158,11,.08);
  border: 1px solid rgba(245,158,11,.3);
  border-radius: 10px;
  padding: 10px 14px;
  margin-bottom: 12px;
}
/* 标题格式不一致提示框 */
.unify-box {
  font-size: 13px;
  color: #b45309;
  background: rgba(245,158,11,.08);
  border: 1px solid rgba(245,158,11,.3);
  border-radius: 10px;
  padding: 10px 14px;
  margin-bottom: 12px;
}
.unify-box .radio-row { color: var(--text-primary); }
.unify-title { font-weight: 600; margin-bottom: 6px; }
.unify-ex {
  font-size: 12px;
  margin: 2px 0 8px;
  padding-left: 4px;
  word-break: break-all;
}
.ex-raw { color: var(--text-secondary); text-decoration: line-through; }
.ex-norm { color: #b45309; font-weight: 500; }
.radio-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text-primary);
  padding: 6px 0;
  cursor: pointer;
}
.split-input { width: 90px; display: inline-block; }
.panel-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
.btn {
  border: 1px solid var(--border, #e0e0e0);
  background: transparent;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-primary);
}
.btn:hover { background: var(--bg-tertiary, #f5f5f7); }
.btn:disabled { opacity: .5; cursor: not-allowed; }
.btn.primary {
  background: var(--accent, #007aff);
  color: #fff;
  border-color: var(--accent, #007aff);
}
@media (max-width: 720px) {
  .chapter-preview { max-height: 36vh; }
}
</style>
