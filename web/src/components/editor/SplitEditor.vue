<template>
  <div
    ref="rootEl"
    class="reference-panel split-panel"
    :class="['split-' + dir, { 'drop-target': dragOver }]"
    :style="dir === 'h' ? { width: width + 'px' } : { height: heightV + 'px' }"
    @dragover.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop.prevent="onDrop"
    @mouseup="onSelUp"
  >
    <!-- 左缘拖宽（左右分屏·桌面） -->
    <div class="split-resize" title="拖动调整宽度 · 点击互换两页" @pointerdown="startResize"></div>
    <!-- ⇄ 互换钮：骑在两页的缝上（fixed 挂视口坐标，不受面板 overflow 裁剪） -->
    <div class="sr-swap" :style="dir === 'h' ? { right: width - 11 + 'px' } : undefined">⇄</div>
    <!-- 顶部抓手（上下分屏 / 手机端上下抽屉） -->
    <div class="split-grabber" title="拖动调节两屏比例" @pointerdown="startGrab"><span></span></div>
    <div v-if="dragOver" class="drop-hint">📄 松开：在这页打开{{ dragTitle ? '「' + dragTitle + '」' : '' }}</div>

    <div class="panehead" :class="{ 'is-active': activePane }">
      <button class="split-nav" :disabled="!canPrev" title="上一章" @click="navPrev">‹</button>
      <span class="ph-title" :title="paneTitle">{{ paneTitle }}</span>
      <button class="split-nav" :disabled="!canNext" title="下一章" @click="navNext">›</button>
      <span v-if="dBook" class="ro-pill">拆书 · 只读</span>
      <span class="split-status" :class="{ warn: conflict }">{{ statusText }}</span>
      <button class="ph-btn dir-btn" :title="dir === 'h' ? '切成上下分屏' : '切成左右分屏'" @click="$emit('toggle-dir')">{{ dir === 'h' ? '⇅' : '⇉' }}</button>
      <button class="ph-btn" title="收起分屏" @click="$emit('close')">✕</button>
    </div>

    <div class="split-body">
      <div v-if="loading" class="ref-loading">加载中...</div>
      <template v-else-if="(selectedId && splitEditor) || dBook">
        <!-- 章名头与左页 chapter-virtual-heading 同款：同级负 margin 压进纸顶 -->
        <div v-if="headingParts" class="split-heading">
          <span class="vh-dash"></span>
          <span class="vh-title"><span v-if="headingParts.num" class="vh-num">{{ headingParts.num }} </span>{{ headingParts.name }}</span>
          <span class="vh-dash vh-dash-r"></span>
        </div>
        <div class="split-paper" :class="{ 'has-heading': headingParts }">
          <editor-content :editor="splitEditor" />
        </div>
      </template>
      <div v-else class="split-empty">
        <div class="se-icon">📄</div>
        <div class="se-main">从左侧章节树拖一章过来</div>
        <div class="se-sub">或点上方标题选择 · 这页也能直接写</div>
      </div>
    </div>
    <!-- 拆书原文划词收藏（fixed 视口坐标，与 sr-swap 同思路不受裁剪） -->
    <button ref="clipBtn" class="clip-btn" @mousedown.prevent @click="saveClip">收藏到素材</button>
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import { Slice, Fragment } from '@tiptap/pm/model'
import StarterKit from '@tiptap/starter-kit'
import Highlight from '@tiptap/extension-highlight'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
import { documentApi } from '../../core/api'
import { dismantleApi } from '../../core/api'
import { chapterDisplayTitle, flattenChapterOrder } from '../../core/chapterTitle'
import { editorContentFromText, stripContentChapterHeading } from '../../core/editorContent'
import { AIMark, AIStrikethrough } from '../../composables/useAIToolbar'
import { useToast } from '../../composables/useToast'

const props = defineProps<{
  documents: any[]
  activeDocId: number | null
  defaultDocId: number | null
  dir: 'h' | 'v'
  activePane: boolean
}>()

const emit = defineEmits<{
  close: []
  swap: []
  'toggle-dir': []
  'pane-focus': []
  'resize-w': [width: number]
  'resize-h': [height: number]
}>()

const splitEditor = shallowRef<any>(null)
const selectedId = ref<number | null>(null)
const loading = ref(false)
const wordCount = ref(0)
const saveStatus = ref('已保存')
const width = ref(0)
const heightV = ref(0)
const dragOver = ref(false)
const dragTitle = ref('')
const rootEl = ref<HTMLElement | null>(null)
const clipBtn = ref<HTMLElement | null>(null)
const toast = useToast()

/* ── 拆书只读源：dBook 非空时本页是拆书库里的别人的书，selectedId 恒 null ── */
const dBook = ref<{ id: number; title: string; chapters: { idx: number; title: string }[]; idx: number } | null>(null)
// 编辑区可用宽（粗略扣侧栏 240；手机走上下分屏不受影响）
const areaW = () => Math.max(420, window.innerWidth - 240)
// 两页同权：默认对半分；拖过的记忆值优先
try {
  const half = Math.round(areaW() / 2)
  width.value = Number(localStorage.getItem('xzj-split-w2')) || Math.max(360, Math.min(areaW() - 360, half))
  heightV.value = Number(localStorage.getItem('xzj-split-mh')) || Math.round(window.innerHeight * 0.45)
} catch { width.value = 480; heightV.value = 400 }

const statusText = computed(() =>
  dBook.value ? `拆书 ${dBook.value.idx}/${dBook.value.chapters.length} · ${wordCount.value} 字` :
  conflict.value ? '只读 · 本章在左侧' : (wordCount.value ? `${saveStatus.value} · ${wordCount.value} 字` : saveStatus.value)
)

// 同一章两边同时开=两个编辑器互踩（自动保存互相覆盖）——右侧转只读，左侧切走即恢复
const conflict = computed(() => selectedId.value !== null && selectedId.value === props.activeDocId)

// 容器/角色卡不进分屏：卷和设定容器没有正文，character 在左侧是特殊卡片视图
const SKIP_TYPES = ['volume', 'outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline', 'character']
const chapterOptions = computed(() => flattenChapterOrder(props.documents).filter((d: any) => d.type === 'chapter'))
// 页头标题（拆书模式带书名；文档模式无下拉：选章靠侧栏点/拖 + ‹› 翻章）
const paneTitle = computed(() => {
  if (dBook.value) {
    const ch = dBook.value.chapters.find(c => c.idx === dBook.value!.idx)
    return `《${dBook.value.title}》${ch ? ' · ' + ch.title : ''}`
  }
  const p = headingParts.value
  if (!p) return '拖章节到这里'
  return p.num ? `${p.num} ${p.name}` : p.name
})
// 纸面文件名头：拆书=书名+章名；章节=章号(绿)+章名，设定/笔记=文档名
const headingParts = computed(() => {
  if (dBook.value) {
    const ch = dBook.value.chapters.find(c => c.idx === dBook.value!.idx)
    return { num: '', name: `${dBook.value.title} · ${ch?.title || ''}` }
  }
  const d = props.documents.find((x: any) => x.id === selectedId.value)
  if (!d) return null
  if (d.type === 'chapter') {
    const full = chapterDisplayTitle(d, props.documents)
    const m = /^(第[0-9零一二三四五六七八九十百千万两]+章)\s*(.*)$/.exec(String(full).trim())
    if (m) return { num: m[1], name: m[2] }
    return { num: '', name: String(full) }
  }
  return { num: '', name: String(d.title || '') }
})

const chapterIdx = computed(() => chapterOptions.value.findIndex((d: any) => d.id === selectedId.value))
const prevChapter = computed(() => (chapterIdx.value > 0 ? chapterOptions.value[chapterIdx.value - 1] : null))
const nextChapter = computed(() =>
  chapterIdx.value >= 0 && chapterIdx.value < chapterOptions.value.length - 1 ? chapterOptions.value[chapterIdx.value + 1] : null
)
// ‹› 导航：拆书模式在拆书章节间翻，文档模式在本项目章节间翻
const canPrev = computed(() => dBook.value ? dBook.value.idx > 1 : !!prevChapter.value)
const canNext = computed(() => dBook.value ? dBook.value.idx < dBook.value.chapters.length : !!nextChapter.value)
function navPrev() {
  if (dBook.value) { if (dBook.value.idx > 1) loadDChapter(dBook.value.idx - 1) }
  else if (prevChapter.value) selectDoc(prevChapter.value.id)
}
function navNext() {
  if (dBook.value) { if (dBook.value.idx < dBook.value.chapters.length) loadDChapter(dBook.value.idx + 1) }
  else if (nextChapter.value) selectDoc(nextChapter.value.id)
}

let saveTimer: any = null
const isSettingContent = ref(false)
let retryCount = 0
const isOffline = () => !navigator.onLine

function getHTML() { return splitEditor.value ? splitEditor.value.getHTML() : '' }
function getText() {
  const ed = splitEditor.value
  if (!ed) return ''
  const blocks: string[] = []
  ed.state.doc.forEach((node: any) => blocks.push(node.textContent || ''))
  return blocks.join('\n')
}
function countWords(text: string): number { return text.replace(/\n/g, '').length }

onMounted(() => {
  // 初始尺寸上报给父组件（--split-w/--split-mh 让纸张让位），否则默认尺寸下纸要等拖动才让位
  emit('resize-w', width.value)
  emit('resize-h', heightV.value)
  splitEditor.value = new Editor({
    content: '',
    editable: false,
    extensions: [
      StarterKit,
      Highlight.configure({ multicolor: true }),
      Table.configure({ resizable: true }),
      TableRow, TableHeader, TableCell,
      AIMark, AIStrikethrough,
    ],
    // 同主编辑器：粘贴强制纯文本+按行成段（防 VS Code 复制来的硬换行被吃成空格）
    editorProps: {
      handlePaste: (view: any, event: ClipboardEvent) => {
        const text = event.clipboardData?.getData('text/plain')
        if (!text) return false
        const lines = text.replace(/\r\n?/g, '\n').split('\n')
        while (lines.length && !lines[lines.length - 1].trim()) lines.pop()
        if (!lines.length) return true
        const { state } = view
        const paras = lines.map((l: string) =>
          state.schema.nodes.paragraph.create(null, l.length ? [state.schema.text(l)] : undefined)
        )
        const slice = new Slice(Fragment.fromArray(paras), 1, 1)
        view.dispatch(state.tr.replaceSelection(slice))
        return true
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (isSettingContent.value) return
      wordCount.value = countWords(getText())
      saveStatus.value = '编辑中...'
      if (selectedId.value) {
        try { localStorage.setItem('draft-' + selectedId.value, ed.getHTML()) } catch {}
      }
      if (saveTimer) clearTimeout(saveTimer)
      saveTimer = setTimeout(() => autoSave(), 1500)
    },
  })
  // 焦点即「活动页」——侧栏点章会开进最后聚焦的那页
  splitEditor.value.on('focus', () => emit('pane-focus'))
})

watch(conflict, (c) => {
  try { splitEditor.value?.setEditable(!c && !!selectedId.value) } catch {}
})

onBeforeUnmount(() => {
  // 卸载前冲刷未落盘的编辑（1.5s 防抖窗口内的字）
  if (saveTimer) { clearTimeout(saveTimer); autoSave() }
  splitEditor.value?.destroy()
})

async function selectDoc(id: number | null | undefined) {
  if (!id || id === selectedId.value) return
  hasPicked = true
  // 换文档前先冲刷旧文档的待保存编辑（同主编辑器 openDoc 的教训：只清 timer 会丢 1.5s 的字）
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; await autoSave() }
  dBook.value = null // 离开拆书模式
  hideClip()
  selectedId.value = id
  await loadDoc(id)
}

/* ── 拆书只读源 ── */
async function openDismantle(bookId: number, idx = 1) {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; await autoSave() }
  loading.value = true
  try {
    const res: any = await dismantleApi.get(bookId)
    const d = res.data?.data
    if (!d?.book) return
    selectedId.value = null
    dBook.value = { id: bookId, title: String(d.book.title || ''), chapters: d.toc || [], idx }
    await loadDChapter(idx)
  } catch {
    toast.error('拆书加载失败')
  } finally {
    loading.value = false
  }
}
async function loadDChapter(idx: number) {
  const b = dBook.value
  if (!b || !splitEditor.value) return
  loading.value = true
  hideClip()
  let content = ''
  try {
    const res: any = await dismantleApi.chapter(b.id, idx)
    content = String(res.data?.data?.content || '')
    try { localStorage.setItem(`xzj-dismantle-${b.id}-${idx}`, content) } catch {}
  } catch {
    try { content = localStorage.getItem(`xzj-dismantle-${b.id}-${idx}`) || '' } catch {}
  }
  isSettingContent.value = true
  splitEditor.value.setEditable(false)
  splitEditor.value.commands.setContent(editorContentFromText(content), { emitUpdate: false })
  b.idx = idx
  wordCount.value = countWords(getText())
  saveStatus.value = '只读'
  setTimeout(() => { isSettingContent.value = false }, 100)
  loading.value = false
}

/* ── 划词收藏（仅拆书模式）── */
function onSelUp() {
  if (!dBook.value) return
  const sel = window.getSelection()
  const txt = sel ? sel.toString().trim() : ''
  if (!sel || !sel.rangeCount || txt.length < 4 || !rootEl.value?.contains(sel.anchorNode)) { hideClip(); return }
  const r = sel.getRangeAt(0).getBoundingClientRect()
  const btn = clipBtn.value
  if (!btn) return
  btn.style.display = 'block'
  btn.style.left = Math.min(window.innerWidth - 120, Math.max(8, r.left + r.width / 2 - 55)) + 'px'
  btn.style.top = Math.max(8, r.top - 40) + 'px'
}
function hideClip() { if (clipBtn.value) clipBtn.value.style.display = 'none' }
async function saveClip() {
  const b = dBook.value
  const sel = window.getSelection()?.toString().trim() || ''
  if (!b || !sel) return
  try {
    await dismantleApi.addClip(b.id, { chapter_idx: b.idx, text: sel })
    toast.success('已收藏到拆书素材')
  } catch {
    toast.error('收藏失败，请重试')
  }
  hideClip()
  window.getSelection()?.removeAllRanges()
}

async function loadDoc(id: number) {
  const doc = props.documents.find((x: any) => x.id === id)
  if (!doc || !splitEditor.value) return
  // 只开可编辑正文：卷/设定容器/角色卡不进分屏
  if (SKIP_TYPES.includes(doc.type)) { selectedId.value = null; return }
  loading.value = true
  try {
    let content = ''
    try {
      const res = await documentApi.get(id)
      const docData = res.data?.data || res.data
      content = (typeof docData?.content === 'string') ? docData.content : ''
      try { localStorage.setItem('xzj-doc-content-' + id, content) } catch {}
    } catch {
      try { content = localStorage.getItem('xzj-doc-content-' + id) || '' } catch {}
    }
    if (doc.type === 'chapter') content = stripContentChapterHeading(content, doc.title)
    isSettingContent.value = true
    const html = content.trim().startsWith('<') ? content : null
    if (html) {
      splitEditor.value.commands.setContent(html, { emitUpdate: false })
    } else {
      splitEditor.value.commands.setContent(editorContentFromText(content), { emitUpdate: false })
    }
    wordCount.value = countWords(getText())
    saveStatus.value = '已保存'
    splitEditor.value.setEditable(!conflict.value)
    setTimeout(() => { isSettingContent.value = false }, 100)
  } finally {
    loading.value = false
  }
}

async function autoSave() {
  const id = selectedId.value
  if (!id || !splitEditor.value) return
  if (conflict.value) return // 只读态不可能有编辑
  if (isOffline()) {
    saveStatus.value = '📡 离线中，已存本地'
    return
  }
  try {
    saveStatus.value = '保存中...'
    await documentApi.update(id, { content: getHTML() })
    saveStatus.value = '已保存'
    retryCount = 0
    try { localStorage.removeItem('draft-' + id); localStorage.setItem('xzj-doc-content-' + id, getHTML()) } catch {}
  } catch {
    if (retryCount < 3) {
      retryCount++
      saveStatus.value = `保存失败，重试(${retryCount}/3)...`
      setTimeout(() => autoSave(), 2000 * retryCount)
    } else {
      saveStatus.value = '⚠️ 保存失败，已存本地'
      retryCount = 0
    }
  }
}

/* 左右分屏拖宽（对称夹取：两页都保底 360）；位移<6px 视为点击=互换两页 */
function startResize(e: PointerEvent) {
  const target = e.currentTarget as HTMLElement
  target.setPointerCapture(e.pointerId)
  const startX = e.clientX
  const startW = width.value
  let dragged = false
  const onMove = (ev: PointerEvent) => {
    if (!dragged && Math.abs(ev.clientX - startX) < 6) return // 阈值内不动，留给点击
    dragged = true
    width.value = Math.min(areaW() - 360, Math.max(360, startW + (startX - ev.clientX)))
    emit('resize-w', width.value)
  }
  const onUp = () => {
    target.removeEventListener('pointermove', onMove)
    target.removeEventListener('pointerup', onUp)
    if (!dragged) { emit('swap'); return } // 原地点击=两页互换
    try { localStorage.setItem('xzj-split-w2', String(width.value)) } catch {}
  }
  target.addEventListener('pointermove', onMove)
  target.addEventListener('pointerup', onUp)
}

/* 上下分屏拖高（30%-70% 屏高），左右方向记忆互相独立 */
function startGrab(e: PointerEvent) {
  const target = e.currentTarget as HTMLElement
  target.setPointerCapture(e.pointerId)
  const onMove = (ev: PointerEvent) => {
    heightV.value = Math.min(window.innerHeight * 0.7, Math.max(window.innerHeight * 0.3, window.innerHeight - ev.clientY))
    emit('resize-h', heightV.value)
  }
  const onUp = () => {
    target.removeEventListener('pointermove', onMove)
    target.removeEventListener('pointerup', onUp)
    try { localStorage.setItem('xzj-split-mh', String(Math.round(heightV.value))) } catch {}
  }
  target.addEventListener('pointermove', onMove)
  target.addEventListener('pointerup', onUp)
}

/* 侧栏拖章到这页：dataTransfer text/plain = doc id（DocTreeNode 现成） */
function onDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dragOver.value = true
}
function onDragLeave() { dragOver.value = false }
function onDrop(e: DragEvent) {
  dragOver.value = false
  const id = Number(e.dataTransfer?.getData('text/plain'))
  if (!id) return
  const doc = props.documents.find((x: any) => x.id === id)
  if (!doc || SKIP_TYPES.includes(doc.type)) return
  if (id === props.activeDocId) return // 已在左侧——拖过来只会转只读，别折腾
  selectDoc(id)
}

// 首次打开：默认定位到上一章（对照续写是最常见场景）；用户自己选过章后不再自动跳
let hasPicked = false
watch(() => props.defaultDocId, (id) => {
  if (!hasPicked && selectedId.value === null && id && id !== props.activeDocId) {
    selectDoc(id)
  }
}, { immediate: true })

defineExpose({ openDoc: selectDoc, getSelected: () => selectedId.value, openDismantle })
</script>

<style scoped>
.split-panel { display: flex; flex-direction: column; overflow: hidden; }

/* 页头：单行（活动点·章选择·状态·方向·关闭），方案二对等双栏形态 */
.panehead {
  display: flex; align-items: center; gap: 6px;
  height: 32px; /* 与左页 left-pane-head 同高——两页章名头 Top 对齐 */
  padding: 0 10px;
  border-bottom: 1px solid var(--ed-line, #e2d8c2);
  background: var(--ed-panel, #f8f3e8);
  font-size: 13px;
  flex-shrink: 0;
  position: relative;
}
/* 活动页=页头底部品牌绿细线（比圆点醒目），文字也提亮 */
.panehead.is-active::after {
  content: '';
  position: absolute; left: 0; right: 0; bottom: -1px;
  height: 2px;
  background: linear-gradient(90deg, transparent, #6aa877 18%, #6aa877 82%, transparent);
}
.panehead.is-active .ph-title { color: var(--ed-ink, #33302a); }
.panehead .ph-title { color: var(--ed-ink-2, #8a8272); }
.split-nav {
  border: 1px solid var(--ed-line, #e2d8c2);
  background: var(--ed-panel-2, rgba(255,255,255,.55));
  color: var(--ed-ink, #33302a);
  width: 24px; height: 24px; border-radius: 6px;
  cursor: pointer; font-size: 14px; line-height: 1;
  flex: 0 0 auto; display: inline-flex; align-items: center; justify-content: center;
}
.split-nav:disabled { opacity: .35; cursor: default; }
.ph-title {
  flex: 1; min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--ed-ink, #33302a);
  text-align: center;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  user-select: none;
}
.split-status {
  font-size: 11.5px; color: var(--ed-ink-2, #8a8272);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  max-width: 40%;
}
.split-status.warn { color: #c77b2a; }
.ph-btn {
  border: none; background: none; cursor: pointer;
  font-size: 13px; color: var(--ed-ink-2, #8a8272);
  width: 24px; height: 24px; border-radius: 6px;
  flex: 0 0 auto;
}
.ph-btn:hover { background: rgba(129,187,142,.15); color: #6aa877; }
/* 拆书只读标（金字 pill，与 demo 定稿一致） */
.ro-pill {
  font-size: 10px; font-weight: 700; color: #a8842e;
  background: rgba(201,163,78,.14); border-radius: 99px; padding: 1.5px 8px;
  flex: 0 0 auto; white-space: nowrap;
}
/* 划词收藏浮钮（fixed 视口坐标） */
.clip-btn {
  position: fixed; z-index: 340; display: none;
  background: var(--ed-ink, #33302a); color: #fff; border: none; border-radius: 8px;
  padding: 6px 12px; font-size: 12px; cursor: pointer;
  box-shadow: 0 4px 14px rgba(60,45,20,.3);
}
.clip-btn:hover { background: #6aa877; }

/* 拖放目标高亮 + 提示 */
.drop-target { box-shadow: inset 0 0 0 2px #6aa877; }
.drop-target .split-empty { border-color: #6aa877; background: rgba(129,187,142,.08); }
.drop-hint {
  position: absolute; z-index: 30;
  left: 50%; top: 50%; transform: translate(-50%, -50%);
  background: rgba(106,168,119,.95); color: #fff;
  padding: 10px 18px; border-radius: 10px;
  font-size: 14px; pointer-events: none;
  box-shadow: 0 8px 24px rgba(60,45,20,.25);
}

/* 左右分屏：左缘拖宽（上下分屏/手机隐藏，见下方） */
.split-resize {
  position: absolute;
  left: 0; top: 0; bottom: 0;
  width: 7px;
  cursor: col-resize;
  z-index: 5;
  display: none;
}
.split-h .split-resize { display: block; }
.split-resize:hover { background: linear-gradient(90deg, rgba(129,187,142,.45), transparent); }
/* ⇄ 互换钮骑缝：fixed 定位（width 响应式算 right），悬停分隔条时浮出；
   pointer-events:none 让点击落到缝的拖动条上（同一套互换逻辑） */
.sr-swap {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  font-size: 12px; color: #6aa877;
  background: var(--ed-paper, #fbf7ee);
  border: 1px solid var(--ed-line, #e2d8c2);
  box-shadow: 0 1px 6px rgba(60,45,20,.18);
  border-radius: 99px; width: 22px; height: 22px;
  display: flex; align-items: center; justify-content: center;
  opacity: 0; transition: opacity .15s;
  pointer-events: none;
  z-index: 320;
}
.split-h .split-resize:hover ~ .sr-swap { opacity: .95; }
@media (max-width: 768px) { .sr-swap { display: none; } }
/* 上下分屏抓手（上下方向或手机端显示） */
.split-grabber {
  display: none;
  padding: 5px 0 1px;
  cursor: row-resize;
  flex-shrink: 0;
}
.split-v .split-grabber { display: block; }
.split-grabber span {
  display: block;
  width: 38px; height: 4px;
  border-radius: 2px;
  background: var(--ed-line, #e2d8c2);
  margin: 0 auto;
}
.split-grabber:hover span { background: #6aa877; }
@media (max-width: 768px) {
  .split-resize { display: none !important; }
  .split-grabber { display: block; }
}

.split-body {
  flex: 1;
  overflow-y: auto;
  /* 与左页 .tiptap-container 同款桌面与留白 */
  background: linear-gradient(165deg, var(--ed-desk-a, #f2ebdb) 0%, var(--ed-desk-b, #ece1cb) 100%);
  padding: 56px 24px 96px;
}
.ref-loading { padding: 16px; text-align: center; color: var(--ed-ink-2, #8a8272); font-size: 13px; }
.split-empty {
  max-width: 400px;
  margin: 48px auto 0;
  border: 1.5px dashed var(--ed-dash, #c9bfa6);
  border-radius: 14px;
  padding: 44px 24px;
  text-align: center;
  background: rgba(255, 255, 255, .28);
}
.split-empty .se-icon { font-size: 30px; margin-bottom: 10px; opacity: .8; }
.split-empty .se-main { font-size: 14px; color: var(--ed-ink, #33302a); line-height: 1.9; }
.split-empty .se-sub { font-size: 12px; color: var(--ed-ink-2, #8a8272); margin-top: 4px; }
/* 右页纸张：与左页 .ProseMirror 纸张卡同款（桌面+居中纸卡+双线纸边），窄页里 padding 收窄 */
.split-paper {
  max-width: var(--paper-w, 820px);
  margin: 0 auto;
  box-sizing: border-box;
  min-height: 62vh;
  padding: 52px 60px calc(72px + var(--ds-bottom-pad, 0px));
  background: var(--ed-paper, #fbf7ee);
  border: 1px solid var(--ed-paper-line, #e6dcc6);
  border-radius: 6px;
  outline: 1px solid var(--ed-paper-outline, rgba(230,220,198,.55));
  outline-offset: -9px;
  box-shadow: 0 2px 14px var(--ed-shadow, rgba(96,74,40,.10));
}
/* 章名头=左页 chapter-virtual-heading 同款（21px/字距2px/64px 渐隐线/负 margin 压纸顶） */
.split-heading {
  position: relative;
  z-index: 2;
  pointer-events: none;
  max-width: var(--paper-w, 820px);
  margin: 0 auto -42px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}
.split-heading .vh-dash { flex: 0 0 64px; height: 1px; background: linear-gradient(90deg, transparent, var(--ed-dash, #c9bfa6)); }
.split-heading .vh-dash-r { background: linear-gradient(90deg, var(--ed-dash, #c9bfa6), transparent); }
.split-heading .vh-title { font-size: 21px; font-weight: 600; line-height: 30px; letter-spacing: 2px; color: var(--ed-ink, #4a4438); }
.split-heading .vh-num { color: #6aa877; }
/* 右侧正文：字号/行距跟随显示设置，跟随 --ed-* 主题 token */
.split-paper.has-heading { padding-top: 66px; }
.split-paper :deep(.ProseMirror) {
  outline: none;
  /* 与主编辑器正文同源：衬线字体+17px 基准（editor-page.css 基础规则同款） */
  font-family: var(--font-serif);
  font-size: var(--ds-font-size, 17px);
  line-height: var(--ds-line-height, 1.9);
  letter-spacing: var(--ds-letter-spacing, normal);
  color: var(--ed-ink, #33302a);
  min-height: 200px;
}
.split-paper :deep(.ProseMirror p) { margin: 0.6em 0; }
.split-paper :deep(.ProseMirror h1) { font-size: 1.4em; font-weight: 700; margin: .8em 0 .4em; }
.split-paper :deep(.ProseMirror h2) { font-size: 1.25em; font-weight: 600; margin: .7em 0 .35em; }
.split-paper :deep(.ProseMirror h3) { font-size: 1.1em; font-weight: 600; margin: .6em 0 .3em; }
.split-paper :deep(.ProseMirror blockquote) {
  border-left: 3px solid var(--ed-line, #e2d8c2);
  padding-left: 12px; margin: .5em 0;
  color: var(--ed-ink-2, #8a8272);
}
.split-paper :deep(.ProseMirror ul), .split-paper :deep(.ProseMirror ol) { padding-left: 1.5em; margin: .4em 0; }
.split-paper :deep(.ProseMirror table) { border-collapse: collapse; margin: .5em 0; }
.split-paper :deep(.ProseMirror th), .split-paper :deep(.ProseMirror td) { border: 1px solid var(--ed-line, #e2d8c2); padding: 4px 8px; font-size: .9em; }

/* 手机端紧凑化（手机恒为上下分屏） */
@media (max-width: 768px) {
  .panehead { padding: 4px 8px; }
  .dir-btn { display: none; } /* 手机恒为上下抽屉，不给切方向 */
  .split-body { padding: 12px 12px 48px; }
  .split-paper { padding: 26px 20px 64px; min-height: 40vh; }
  /* 手机章名头=左页手机版参数：常规流、17px */
  .split-heading { position: static; margin: 0.3em 0 1em; max-width: none; }
  .split-heading .vh-dash { flex: 0 0 38px; }
  .split-heading .vh-title { font-size: 17px; line-height: 24px; letter-spacing: 1.5px; }
}
</style>
