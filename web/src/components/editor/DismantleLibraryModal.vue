<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="dl-modal" @click.stop>
      <!-- ═══ 库列表 ═══ -->
      <template v-if="!reportBook">
        <div class="dl-head">
          <svg class="ic" viewBox="0 0 19 19"><path d="M3 4c2-1 4.5-1 6.5.5C11.5 3 14 3 16 4v10.5c-2-1-4.5-1-6.5.5-2-1.5-4.5-1.5-6.5-.5z"/><path d="M9.5 4.5v10.5"/></svg>
          <b>拆书库</b>
          <span class="dl-sub">{{ books.length }}/5 本 · 只做参考，不写入你的创作内容</span>
          <button class="dl-x" @click="close">✕</button>
        </div>
        <div class="dl-tabs">
          <button :class="{ on: tab === 'lib' }" @click="tab = 'lib'">书库</button>
          <button :class="{ on: tab === 'clips' }" @click="switchClips">素材（{{ clips.length }}）</button>
        </div>

        <!-- 书库 -->
        <div v-if="tab === 'lib'" class="dl-body">
          <div v-if="uploading.step !== 'idle'" class="dl-up">
            <template v-if="uploading.step === 'parse'">
              <div class="dl-up-txt">正在分章…</div>
            </template>
            <template v-else-if="uploading.step === 'post'">
              <div class="dl-up-txt">正在上传 {{ uploading.chapters }} 章 / {{ uploading.chars }} 字…</div>
            </template>
          </div>
          <button v-else class="dl-add" @click="fileInput?.click()">
            <svg class="ic" viewBox="0 0 19 19"><path d="M9.5 14V4M5.5 8l4-4 4 4M3.5 15.5h12"/></svg>
            拆新书 · 上传 txt（自动分章，全程不经过 AI 存储）
          </button>
          <input ref="fileInput" type="file" accept=".txt" style="display:none" @change="onPick" />

          <div v-if="!books.length && uploading.step === 'idle'" class="dl-empty">
            还没拆过书。上传一本你想学的书，<br />拆出大纲 / 人物卡 / 节奏 / 套路，在分屏里对照着读。
          </div>

          <div v-for="b in books" :key="b.id" class="dl-card" :class="{ extracting: b.status === 'extracting' }">
            <div class="dl-book" :style="{ background: coverColor(b.id) }"></div>
            <div class="dl-card-t">
              <b>{{ b.title }}</b>
              <span>
                <template v-if="b.status === 'extracting'">拆解中 {{ b.progress }}%</template>
                <template v-else-if="b.status === 'failed'">拆解失败：{{ b.error || '重试请删除后重拆' }}</template>
                <template v-else>{{ fmtWords(b.words) }} · {{ b.chapters }} 章 · {{ fmtDate(b.created_at) }}</template>
              </span>
              <div v-if="b.status === 'extracting'" class="dl-bar"><i :style="{ width: b.progress + '%' }"></i></div>
            </div>
            <div class="dl-card-acts">
              <button v-if="b.status === 'done'" class="dl-btn" @click="openReport(b)">报告</button>
              <button v-if="b.status === 'done'" class="dl-btn pri" @click="openSplit(b)">分屏打开</button>
              <button class="dl-btn del" @click="del(b)">删除</button>
            </div>
          </div>
        </div>

        <!-- 素材 -->
        <div v-else class="dl-body">
          <div v-if="!clips.length" class="dl-empty">在分屏里划选拆书原文 → 点「收藏」，素材会汇到这里。</div>
          <div v-for="cp in clips" :key="cp.id" class="dl-clip">
            <div class="dl-clip-t">{{ cp.text }}</div>
            <div class="dl-clip-meta">《{{ cp.book_title }}》第{{ cp.chapter_idx }}章 · {{ fmtDate(cp.created_at) }}
              <button class="dl-mini" @click="copyClip(cp)">复制</button>
              <button class="dl-mini del" @click="delClip(cp)">删除</button>
            </div>
          </div>
        </div>
      </template>

      <!-- ═══ 报告 ═══ -->
      <template v-else>
        <div class="dl-head">
          <button class="dl-back" @click="reportBook = null">‹ 返回</button>
          <b>{{ reportBook.title }} · 拆解报告</b>
          <button class="dl-x" @click="close">✕</button>
        </div>
        <div class="dl-body">
          <div class="dl-out-tabs">
            <button v-for="k in outKinds" :key="k.k" :class="{ on: outTab === k.k }" @click="outTab = k.k">{{ k.t }}</button>
          </div>
          <pre class="dl-out">{{ report.outputs[outTab] || '（无内容）' }}</pre>
          <div v-if="outTab === 'anno' && report.annotations.length" class="dl-annos">
            <div v-for="(a, i) in report.annotations" :key="i" class="dl-anno">
              <span class="dl-anno-k" :data-k="a.kind">{{ a.kind }}</span>
              <b>第{{ a.chapter_idx }}章 · {{ a.title }}</b>
              <span class="dl-anno-n">{{ a.note }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { dismantleApi } from '../../core/api'

const props = defineProps<{ modelValue: boolean; projectId: number }>()
const emit = defineEmits<{ close: []; 'open-split': [book: any] }>()

const tab = ref<'lib' | 'clips'>('lib')
const books = ref<any[]>([])
const clips = ref<any[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref({ step: 'idle' as 'idle' | 'parse' | 'post', chapters: 0, chars: 0 })
const reportBook = ref<any>(null)
const report = ref<any>({ outputs: {}, annotations: [], toc: [] })
const outTab = ref('outline')
const outKinds = [
  { k: 'outline', t: '大纲' }, { k: 'chars', t: '人物' }, { k: 'net', t: '关系' },
  { k: 'pulse', t: '节奏' }, { k: 'trick', t: '套路' }, { k: 'anno', t: '标注' },
]

let pollTimer: any = null

watch(() => props.modelValue, (v) => { if (v) { reportBook.value = null; tab.value = 'lib'; load() } }, { immediate: true })
// Editor 的 projectId 由 public id 异步解析，弹层先开时可能还是 0——解析好后补拉一次
watch(() => props.projectId, (v, old) => { if (props.modelValue && v > 0 && old !== v) load() })

async function load() {
  try {
    const res = await dismantleApi.list(props.projectId)
    books.value = res.data?.data || []
    schedulePoll()
  } catch { /* 静默 */ }
}
function schedulePoll() {
  clearTimeout(pollTimer)
  if (books.value.some((b: any) => b.status === 'extracting')) {
    pollTimer = setTimeout(async () => {
      try {
        const res = await dismantleApi.list(props.projectId)
        books.value = res.data?.data || []
      } catch {}
      schedulePoll()
    }, 3000)
  }
}
onBeforeUnmount(() => clearTimeout(pollTimer))

async function switchClips() {
  tab.value = 'clips'
  try {
    const res = await dismantleApi.listClips(props.projectId)
    clips.value = res.data?.data || []
  } catch {}
}

/* ── 上传 + 分章（复用导入建书的正则，删掉卷处理/归一，拆书只要平铺章节） ── */
const CHAPTER_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回话](?:[^，。！？\n]{0,40}[。！？]?)?|第\s*[0-9]{1,4}\s*[、.．]\s*[^\s第][^\n]{0,78}|序章|楔子|引子|尾声|终章|番外[^\n]{0,30}|Chapter\s+\d+[^\n]{0,40}|[0-9]{1,4}[、.．]\s*[^\s第][^\n]{0,38})[ \t　]*$/

function parseChapters(text: string): { title: string; content: string }[] {
  const lines = text.split(/\r\n|\r|\n/)
  const out: { title: string; content: string }[] = []
  let cur: { title: string; content: string } | null = null
  for (const line of lines) {
    const t = line.trim()
    if (t && CHAPTER_RE.test(t)) {
      if (cur && cur.content.trim()) out.push(cur)
      cur = { title: t.replace(/\s+/g, ' '), content: '' }
    } else if (cur) {
      cur.content += line + '\n'
    } else if (t) {
      // 章节标题前的前言：并进第一章
      cur = { title: '开篇', content: line + '\n' }
    }
  }
  if (cur && cur.content.trim()) out.push(cur)
  // 没识别到标题 → 3000 字一段切
  if (!out.length && text.trim()) {
    const paras = text.split(/\n\s*\n/)
    let buf = ''
    let n = 1
    for (const p of paras) {
      buf += p + '\n\n'
      if (buf.length > 3000) { out.push({ title: `第${n}节`, content: buf }); buf = ''; n++ }
    }
    if (buf.trim()) out.push({ title: `第${n}节`, content: buf })
  }
  return out
}

async function readTextFile(file: File): Promise<string> {
  const buf = await file.arrayBuffer()
  try { return new TextDecoder('utf-8', { fatal: true }).decode(buf) } catch {}
  try { return new TextDecoder('gbk').decode(buf) } catch {}
  return new TextDecoder('utf-8').decode(buf)
}

async function onPick(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  uploading.value = { step: 'parse', chapters: 0, chars: 0 }
  try {
    const text = await readTextFile(f)
    const chapters = parseChapters(text)
    if (!chapters.length) { alert('没读到内容，换一个 txt 试试'); uploading.value.step = 'idle'; return }
    const title = f.name.replace(/\.txt$/i, '').slice(0, 60) || '未命名拆书'
    uploading.value = { step: 'post', chapters: chapters.length, chars: text.length }
    await dismantleApi.create({ project_id: props.projectId, title, chapters })
    uploading.value.step = 'idle'
    await load()
  } catch (err: any) {
    alert('上传失败：' + (err?.response?.data?.error || err?.message || '未知错误'))
    uploading.value.step = 'idle'
  }
  if (fileInput.value) fileInput.value.value = ''
}

async function openReport(b: any) {
  try {
    const res = await dismantleApi.get(b.id)
    report.value = res.data?.data || { outputs: {}, annotations: [], toc: [] }
    reportBook.value = b
    outTab.value = 'outline'
  } catch {}
}
function openSplit(b: any) { emit('open-split', b); close() }
async function del(b: any) {
  if (!confirm(`删除《${b.title}》的拆解（含原文、报告、素材）？`)) return
  await dismantleApi.remove(b.id).catch(() => {})
  await load()
}
async function delClip(cp: any) {
  await dismantleApi.removeClip(cp.id).catch(() => {})
  clips.value = clips.value.filter((x: any) => x.id !== cp.id)
}
function copyClip(cp: any) {
  navigator.clipboard?.writeText(cp.text).catch(() => {})
}

function close() { emit('close') }
function fmtWords(n: number) { return n >= 10000 ? (n / 10000).toFixed(1) + ' 万字' : n + ' 字' }
function fmtDate(s: string) { return String(s || '').slice(5, 10) }
function coverColor(id: number) {
  const cs = ['linear-gradient(150deg,#3d5a80,#98c1d9)', 'linear-gradient(150deg,#6d597a,#b5a642)', 'linear-gradient(150deg,#3a5a40,#74a892)', 'linear-gradient(150deg,#9a6d5d,#d8a48f)', 'linear-gradient(150deg,#5f747e,#a3bcc9)']
  return cs[id % cs.length]
}
</script>

<style scoped>
.dl-modal {
  background: var(--ed-panel, #f8f3e8);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(40,30,10,.25);
  width: 460px; max-width: 92vw;
  max-height: 84vh;
  display: flex; flex-direction: column;
  overflow: hidden;
  color: var(--ed-ink, #33302a);
}
.dl-head { display: flex; align-items: center; gap: 8px; padding: 14px 16px 10px; border-bottom: 1px solid var(--ed-line, #e2d8c2); flex: none; }
.dl-head b { font-size: 15px; }
.dl-head .ic { width: 17px; height: 17px; stroke: #6aa877; stroke-width: 1.6; fill: none; stroke-linecap: round; stroke-linejoin: round; }
.dl-sub { font-size: 10.5px; color: var(--ed-ink-2, #8a8272); }
.dl-x { margin-left: auto; border: none; background: none; cursor: pointer; color: var(--ed-ink-2, #8a8272); font-size: 13px; }
.dl-back { border: none; background: none; cursor: pointer; color: #6aa877; font-size: 13px; font-weight: 600; padding: 0; }
.dl-tabs { display: flex; gap: 6px; padding: 10px 16px 0; flex: none; }
.dl-tabs button { border: none; background: rgba(0,0,0,.04); color: var(--ed-ink-2, #8a8272); border-radius: 8px; padding: 6px 14px; font-size: 12px; cursor: pointer; }
.dl-tabs button.on { background: rgba(106,168,119,.14); color: #6aa877; font-weight: 700; }
.dl-body { padding: 12px 16px 16px; overflow-y: auto; flex: 1; min-height: 260px; }

.dl-add { width: 100%; border: 1.5px dashed var(--ed-dash, #c9bfa6); background: none; border-radius: 10px; padding: 12px; font-size: 12px; color: var(--ed-ink-2, #8a8272); cursor: pointer; display: flex; align-items: center; justify-content: center; gap: 7px; margin-bottom: 10px; }
.dl-add:hover { border-color: #6aa877; color: #6aa877; }
.dl-add .ic { width: 15px; height: 15px; stroke: currentColor; stroke-width: 1.6; fill: none; stroke-linecap: round; }
.dl-up { border: 1.5px dashed var(--ed-dash, #c9bfa6); border-radius: 10px; padding: 12px; text-align: center; font-size: 12px; color: var(--ed-ink-2, #8a8272); margin-bottom: 10px; }

.dl-empty { text-align: center; color: var(--ed-ink-2, #8a8272); font-size: 12px; line-height: 2; padding: 30px 10px; }

.dl-card { display: flex; align-items: center; gap: 10px; padding: 10px; border: 1px solid var(--ed-line, #e2d8c2); background: rgba(255,255,255,.5); border-radius: 12px; margin-bottom: 8px; }
.dl-card.extracting { border-color: #cfe6d3; background: rgba(106,168,119,.06); }
.dl-book { width: 26px; height: 36px; border-radius: 4px; flex: none; }
.dl-card-t { flex: 1; min-width: 0; }
.dl-card-t b { font-size: 13px; display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dl-card-t span { font-size: 10.5px; color: var(--ed-ink-2, #8a8272); }
.dl-bar { height: 4px; background: rgba(0,0,0,.07); border-radius: 99px; overflow: hidden; margin-top: 4px; }
.dl-bar i { display: block; height: 100%; background: #6aa877; border-radius: 99px; transition: width .5s; }
.dl-card-acts { display: flex; gap: 5px; flex: none; }
.dl-btn { border: 1px solid var(--ed-line, #e2d8c2); background: rgba(255,255,255,.6); color: var(--ed-ink, #33302a); border-radius: 7px; padding: 5px 9px; font-size: 11px; font-weight: 600; cursor: pointer; }
.dl-btn.pri { background: #6aa877; border-color: #6aa877; color: #fff; }
.dl-btn.del { color: #b0722f; }

.dl-clip { border: 1px solid var(--ed-line, #e2d8c2); background: rgba(255,255,255,.5); border-radius: 10px; padding: 9px 11px; margin-bottom: 7px; }
.dl-clip-t { font-size: 12px; line-height: 1.8; color: var(--ed-ink, #33302a); }
.dl-clip-meta { font-size: 10px; color: var(--ed-ink-2, #8a8272); margin-top: 5px; display: flex; gap: 8px; align-items: center; }
.dl-mini { border: none; background: rgba(0,0,0,.05); color: var(--ed-ink-2, #8a8272); border-radius: 6px; padding: 2px 8px; font-size: 10px; cursor: pointer; }
.dl-mini.del { color: #b0722f; }

.dl-out-tabs { display: flex; gap: 4px; margin-bottom: 10px; flex-wrap: wrap; }
.dl-out-tabs button { border: none; background: rgba(0,0,0,.04); color: var(--ed-ink-2, #8a8272); border-radius: 8px; padding: 5px 12px; font-size: 11.5px; cursor: pointer; }
.dl-out-tabs button.on { background: rgba(106,168,119,.14); color: #6aa877; font-weight: 700; }
.dl-out { white-space: pre-wrap; font-family: inherit; font-size: 12.5px; line-height: 1.9; color: var(--ed-ink, #33302a); background: rgba(255,255,255,.5); border: 1px solid var(--ed-line, #e2d8c2); border-radius: 10px; padding: 12px 14px; }
.dl-annos { margin-top: 10px; }
.dl-anno { display: flex; gap: 8px; align-items: baseline; padding: 7px 0; border-bottom: 1px solid var(--ed-line-soft, #e8dfc8); font-size: 11.5px; }
.dl-anno b { color: var(--ed-ink, #33302a); flex: none; }
.dl-anno-n { color: var(--ed-ink-2, #8a8272); }
.dl-anno-k { flex: none; font-size: 9.5px; border-radius: 99px; padding: 1.5px 7px; font-weight: 700; background: rgba(106,168,119,.12); color: #6aa877; }
.dl-anno-k[data-k="钩子"] { background: rgba(201,163,78,.15); color: #a8842e; }
.dl-anno-k[data-k="意象"] { background: rgba(109,89,122,.14); color: #6d597a; }
</style>
