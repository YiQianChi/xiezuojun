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
          <div class="ip-fill" :style="{ width: importPct + '%' }"></div>
        </div>
        <div class="ip-count">{{ importedChapters }} / {{ totalChaptersToImport }} 章</div>
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
        <div class="chapter-preview">
          <div v-for="(v, vi) in volumes" :key="vi" class="pv-vol">
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
const importing = ref(false)
// 分批导入进度
const importStatus = ref('')
const importedChapters = ref(0)
const totalChaptersToImport = ref(0)
const importPct = computed(() =>
  totalChaptersToImport.value > 0 ? Math.min(100, Math.round(importedChapters.value / totalChaptersToImport.value * 100)) : 0)

// 章节/卷标题行识别：整行匹配且限制长度，防止误吃正文里提到「第三章」的句子
const VOLUME_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*卷[^\n]{0,40}|卷\s*[0-9零一二三四五六七八九十百千万两]+[^\n]{0,30})[ \t　]*$/
const CHAPTER_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回话][^\n]{0,40}|序章|序|楔子|引子|尾声|终章|番外[^\n]{0,30}|Chapter\s+\d+[^\n]{0,40})[ \t　]*$/

const totalChapters = computed(() => volumes.value.reduce((n, v) => n + v.chapters.length, 0))
const totalChars = computed(() =>
  volumes.value.reduce((n, v) => n + v.chapters.reduce((m, c) => m + countChars(c.content), 0), 0))
const canSubmit = computed(() =>
  bookName.value.trim().length > 0 && (volumes.value.length > 0 ? totalChapters.value > 0 : rawText.value.trim().length > 0))

// 外部（新建弹窗关闭时）调用来重置面板
function reset() {
  step.value = 'pick'
  bookName.value = ''
  rawText.value = ''
  volumes.value = []
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
    const t = line.trim()
    if (t && t.length <= 50 && VOLUME_RE.test(t)) {
      curVol = { title: t, chapters: [] }
      vols.push(curVol)
      curCh = null
      continue
    }
    if (t && t.length <= 50 && CHAPTER_RE.test(t)) {
      foundChapter = true
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
    let vols = volumes.value
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

      importStatus.value = firstBatch ? '创建书籍并导入第一批…' : `导入中（第 ${Math.floor(i / BATCH) + 1} 批）…`
      const res: any = firstBatch
        ? await projectApi.importBook({ name: bookName.value.trim(), volumes: batchVols })
        : await projectApi.importBookBatch({ project_id: projectId!, name: bookName.value.trim(), volumes: batchVols })
      const data = res.data?.data || res.data
      projectId = data.project_id ?? projectId
      importedChapters.value = Math.min(i + slice.length, totalChaptersToImport.value)
      firstBatch = false
    }

    importStatus.value = '导入完成'
    toast.success(`导入完成：${importedChapters.value} 章已上架`)
    emit('update:modelValue', false)
    emit('imported', projectId!)
  } catch (e: any) {
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
