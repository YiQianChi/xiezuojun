<template>
  <div class="sc-overlay" @keydown.esc="onClose">
    <!-- ── 顶栏 ── -->
    <div class="sc-topbar">
      <div class="sc-brand">📐 设定画布</div>
      <div class="sc-seg">
        <button :class="{ on: view === 'board' }" @click="view = 'board'">🗂 白板</button>
        <button :class="{ on: view === 'timeline' }" @click="switchTimeline">🧵 伏笔时间线</button>
      </div>
      <div class="sc-tools" v-if="view === 'board'">
        <button class="sc-tbtn" @click="paletteOpen = !paletteOpen">＋ 加卡片</button>
        <button class="sc-tbtn" @click="addZone">▦ 加分区</button>
        <button class="sc-tbtn" @click="importGraph" :disabled="importing">{{ importing ? '引入中…' : '🔗 引入关系图' }}</button>
        <button class="sc-tbtn" v-if="graphEdges.length" @click="$emit('open-graph')">查看关系图谱 ↗</button>
        <button class="sc-tbtn" @click="addQuoteFromSelection" v-if="getSelection">❝ 摘录选中文字</button>
      </div>
      <div class="sc-spacer"></div>
      <span class="sc-status" :class="{ dirty: saveState !== 'saved' }" @click="flush(true)">{{
        saveState === 'saved' ? '✓ 已保存' : saveState === 'saving' ? '保存中…' : '● 未保存'
      }}</span>
      <div class="sc-zoom" v-if="view === 'board'">
        <button class="sc-tbtn" @click="zoomBy(0.84)">－</button>
        <span class="sc-zpct">{{ Math.round(viewT.scale * 100) }}%</span>
        <button class="sc-tbtn" @click="zoomBy(1.2)">＋</button>
        <button class="sc-tbtn" @click="fitView">适应</button>
      </div>
      <button class="sc-close" @click="onClose" title="收起画布（Esc）">✕</button>
    </div>

    <!-- ═══ 白板视图 ═══ -->
    <div class="sc-board" v-show="view === 'board'" ref="boardEl" :class="{ panning: isPanning }">
      <div class="sc-world" ref="worldEl" :style="worldStyle">
        <svg class="sc-svg">
          <path v-for="(e, i) in graphEdges" :key="'e' + i" :d="e.d" :stroke="e.color" :stroke-dasharray="e.dash"
            class="sc-edge" @click.stop="toast.info('关系线在「关系图谱」里编辑')" />
          <text v-for="(e, i) in graphEdges" :key="'t' + i" :x="e.mx" :y="e.my - 6" text-anchor="middle" class="sc-elabel"
            :fill="e.color">{{ e.label }}</text>
        </svg>

        <!-- 分区（垫底） -->
        <div v-for="z in zones" :key="z.id" class="sc-zone" :style="zoneStyle(z)">
          <div class="sc-zone-bar" @pointerdown.stop="startZoneDrag($event, z)" :title="z.label + '（拖动移动分区）'">
            <span v-if="editingZone !== z.id" class="sc-zone-label" @pointerdown.stop @click.stop="editZoneLabel(z)">{{ z.label }}</span>
            <input v-else v-model="z.label" class="sc-zone-input" @pointerdown.stop @blur="editingZone = null" @keydown.enter="editingZone = null" ref="zoneInputEl" />
            <button class="sc-zone-del" @pointerdown.stop @click.stop="delZone(z)" title="删除分区（卡片不动）">✕</button>
          </div>
        </div>

        <!-- 卡片 -->
        <div v-for="c in cards" :key="c.id" class="sc-card" :class="['k-' + c.kind, { sel: selId === c.id, pinned: c.pinned }]"
          :style="{ left: c.x + 'px', top: c.y + 'px' }" :data-cid="c.id"
          @pointerdown="startCardDrag($event, c)">
          <div class="sc-card-bar">
            <span class="sc-card-kind">{{ kindLabel(c) }}</span>
            <span class="sc-card-acts">
              <button class="sc-act" :class="{ on: c.pinned }" title="钉到右栏「设定」页签置顶" @pointerdown.stop @click.stop="togglePin(c)">{{ c.pinned ? '📌' : '📍' }}</button>
              <button class="sc-act" title="移出画布" @pointerdown.stop @click.stop="removeCard(c)">✕</button>
            </span>
          </div>

          <!-- doc 卡：点开拉正文摘要 -->
          <template v-if="c.kind === 'doc'">
            <div class="sc-card-title" @dblclick="expandDoc(c)">{{ c.title }}</div>
            <div class="sc-card-text" @dblclick="expandDoc(c)">{{ c.desc || excerptOf(c) }}</div>
          </template>

          <!-- graph 卡：关系图角色 -->
          <template v-else-if="c.kind === 'graph'">
            <div class="sc-card-title">{{ c.title }}</div>
            <div class="sc-card-text">{{ c.desc || '关系图角色 · 在关系图谱里编辑' }}</div>
          </template>

          <!-- 摘录卡 / 自由卡：双击可编辑 -->
          <template v-else>
            <div class="sc-card-title" v-if="editing !== c.id" @dblclick="startEdit(c)">{{ c.title || '未命名' }}</div>
            <input v-else v-model="c.title" class="sc-edit-input" @pointerdown.stop placeholder="标题"
              @keydown.enter="focusTa" />
            <div class="sc-card-text quote" v-if="editing !== c.id" @dblclick="startEdit(c)">{{ c.text || '双击填写…' }}</div>
            <textarea v-else v-model="c.text" class="sc-edit-textarea" @pointerdown.stop rows="4" placeholder="内容"
              @keydown.esc.stop="editing = null" @blur="editing = null" ref="editTa"></textarea>
            <div v-if="c.src" class="sc-card-src">📜 {{ c.src }}</div>
          </template>
        </div>
      </div>

      <!-- 空状态 -->
      <div class="sc-empty" v-if="!cards.length && !loading">
        <div class="sc-empty-icon">📐</div>
        <p>画布还空着——把<b>角色卡、设定文档</b>摆上来整理，或<b>引入关系图</b>。</p>
        <p class="sub">卡片可拖拽 / 双击看内容；右栏「设定」页签里钉选的卡会置顶对照。</p>
      </div>

      <!-- 左侧素材抽屉 -->
      <div class="sc-palette" v-if="paletteOpen">
        <div class="sc-palette-head">
          <input v-model="palQuery" placeholder="搜索角色 / 设定…" />
          <button class="sc-tbtn" @click="paletteOpen = false">✕</button>
        </div>
        <div class="sc-palette-body">
          <div class="sc-pal-cap">👥 角色（点一下加到画布）</div>
          <div v-for="d in palCharacters" :key="'pc' + d.id" class="sc-pal-item" @click="addDocCard(d)">
            <b>{{ d.title }}</b><small>{{ d.desc || d.type }}</small>
          </div>
          <div v-if="!palCharacters.length" class="sc-pal-none">左栏「作品相关」里还没建角色卡</div>
          <div class="sc-pal-cap">📄 设定文档</div>
          <div v-for="d in palDocs" :key="'pd' + d.id" class="sc-pal-item" @click="addDocCard(d)">
            <b>{{ d.title }}</b><small>{{ d.type }}</small>
          </div>
          <div v-if="!palDocs.length" class="sc-pal-none">还没有设定文档</div>
          <div class="sc-pal-cap">✚ 其他</div>
          <div class="sc-pal-item" @click="addFreeCard"><b>空白便签</b><small>自由记录</small></div>
        </div>
      </div>

      <div class="sc-hintbar">拖空白平移 · 滚轮缩放 · 拖卡片摆位 · 双击卡片看全文/编辑</div>
    </div>

    <!-- ═══ 伏笔时间线视图 ═══ -->
    <div class="sc-timeline" v-if="view === 'timeline'">
      <div class="tl-head"><div v-for="(ch, i) in chapters" :key="i" class="tl-ch">{{ ch.title }}</div></div>
      <div class="tl-grid"><i v-for="i in chapters.length" :key="'g' + i"></i></div>
      <div class="tl-rows">
        <div v-for="t in threadRows" :key="t.id" class="tl-row" :style="{ top: 56 + t.idx * 78 + 'px' }">
          <div class="tl-rname">{{ t.title }}</div>
          <div class="tl-track">
            <div class="tl-arc" :class="t.state" :style="{ left: t.left + '%', width: Math.max(t.width, 9) + '%' }"
              @click="selThread = t">
              <i class="tl-dot l"></i>
              <span>{{ t.state === 'done' ? `${t.title} · ${chapters[t.from]?.title}→${chapters[t.to]?.title}` : t.title + ' → 进行中' }}</span>
              <i v-if="t.state === 'done'" class="tl-dot r"></i>
            </div>
          </div>
        </div>
        <div v-if="!threadRows.length" class="tl-none">还没有伏笔——正文里选中文字右键「记伏笔」，或去右栏「设定」页签的伏笔清单里加。</div>
      </div>
      <div class="tl-legend">
        <span><i style="background:#6aa877"></i>已回收 {{ tlStats.done }}</span>
        <span><i style="background:#c9a34e"></i>埋设中 {{ tlStats.open }}</span>
        <span><i style="background:#c0504d"></i>超期未收 {{ tlStats.late }}</span>
      </div>
      <!-- 伏笔详情 -->
      <div class="sc-detail" v-if="selThread">
        <button class="sc-detail-x" @click="selThread = null">✕</button>
        <h5>{{ selThread.title }} <span class="tl-tag" :class="selThread.state">{{ tlStateLabel(selThread) }}</span></h5>
        <p v-if="selThread.description">{{ selThread.description }}</p>
        <p class="sub">埋：第 {{ selThread.planted }} 章{{ selThread.to != null ? ` · 收：第 ${selThread.resolved} 章` : ' · <b style="color:#c0504d">还没收</b>' }}</p>
        <p class="sub" v-if="selThread.state === 'late'">埋下已超过 15 章未回收——读者要忘了。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { documentApi, memoryApi, threadApi } from '../../core/api'
import { toast } from '../../composables/useToast'

const props = defineProps<{
  projectId: number
  chapters: { id: number; title: string }[]
  currentChapter?: number
  getSelection?: () => { text: string; chapterTitle: string; chapterNo: number } | null
}>()

const emit = defineEmits<{ close: []; 'open-graph': [] }>()

/* ── 数据模型（整份 JSON 落库） ── */
interface CanvasCard {
  id: string; kind: 'doc' | 'quote' | 'free' | 'graph'
  docId?: number; title: string; text?: string; desc?: string; src?: string
  x: number; y: number; pinned?: boolean
}
interface CanvasZone { id: string; x: number; y: number; w: number; h: number; label: string }
interface CanvasView { x: number; y: number; scale: number }

const cards = ref<CanvasCard[]>([])
const zones = ref<CanvasZone[]>([])
const viewT = ref<CanvasView>({ x: 0, y: 0, scale: 1 })
const view = ref<'board' | 'timeline'>('board')
const loading = ref(true)
const saveState = ref<'saved' | 'dirty' | 'saving'>('saved')
const selId = ref('')
const editing = ref<string | null>(null)
const editingZone = ref<string | null>(null)
const paletteOpen = ref(false)
const palQuery = ref('')
const importing = ref(false)
const isPanning = ref(false)

let uidSeq = Date.now() % 100000
const uid = () => `c${Date.now().toString(36)}${(uidSeq++).toString(36)}`
let saveTimer: ReturnType<typeof setTimeout> | null = null

/* ── 素材库 ── */
const allDocs = ref<any[]>([])
const SETTING_TYPES = ['outline', 'characters', 'worldbuilding', 'chapter_outline', 'note', 'setting']
const palCharacters = computed(() => {
  const q = palQuery.value.trim()
  return allDocs.value.filter((d: any) => d.type === 'character' && (!q || d.title.includes(q)))
    .map((d: any) => ({ id: d.id, title: d.title, type: '角色', desc: descOf(d) }))
})
const palDocs = computed(() => {
  const q = palQuery.value.trim()
  return allDocs.value.filter((d: any) => SETTING_TYPES.includes(d.type) && (!q || d.title.includes(q)))
    .map((d: any) => ({ id: d.id, title: d.title, type: typeLabel(d.type) }))
})
function descOf(d: any): string {
  let meta = d.meta
  if (typeof meta === 'string') { try { meta = JSON.parse(meta) } catch { meta = {} } }
  return String(meta?.desc || meta?.identity || '').slice(0, 80)
}
function typeLabel(t: string): string {
  const m: Record<string, string> = { outline: '大纲', characters: '人物', worldbuilding: '世界观', chapter_outline: '章节大纲', note: '笔记', setting: '设定' }
  return m[t] || '设定'
}

/* ── 卡片正文摘要缓存 ── */
const excerpts = ref<Record<number, string>>({})
function excerptOf(c: CanvasCard): string {
  if (c.docId && excerpts.value[c.docId]) return excerpts.value[c.docId]
  return '双击加载内容…'
}
async function expandDoc(c: CanvasCard) {
  if (!c.docId) return
  if (excerpts.value[c.docId]) return
  try {
    const res = await documentApi.get(c.docId)
    const raw: string = res.data?.data?.content || res.data?.data?.search_text || ''
    const text = raw.split('\n').map((l: string) => l.replace(/^#{1,6}\s*/, '').trim()).filter(Boolean).join('\n')
    excerpts.value[c.docId] = text.slice(0, 400) || '（空文档）'
  } catch { toast.error('内容加载失败') }
}

/* ── 视图变换 ── */
const boardEl = ref<HTMLElement | null>(null)
const worldEl = ref<HTMLElement | null>(null)
const worldStyle = computed(() => ({
  transform: `translate(${viewT.value.x}px, ${viewT.value.y}px) scale(${viewT.value.scale})`,
}))
function toWorld(cx: number, cy: number) {
  const r = boardEl.value!.getBoundingClientRect()
  return { x: (cx - r.left - viewT.value.x) / viewT.value.scale, y: (cy - r.top - viewT.value.y) / viewT.value.scale }
}
function zoomAt(cx: number, cy: number, f: number) {
  const w = toWorld(cx, cy)
  viewT.value.scale = Math.min(2, Math.max(0.3, viewT.value.scale * f))
  const r = boardEl.value!.getBoundingClientRect()
  viewT.value.x = cx - r.left - w.x * viewT.value.scale
  viewT.value.y = cy - r.top - w.y * viewT.value.scale
  markDirty()
}
function zoomBy(f: number) {
  const r = boardEl.value?.getBoundingClientRect()
  if (r) zoomAt(r.left + r.width / 2, r.top + r.height / 2, f)
}
function fitView() { viewT.value = { x: 40, y: 30, scale: 1 }; markDirty() }

/* ── 拖拽：卡片 / 分区 / 平移 ── */
let drag: { type: 'card' | 'zone' | 'pan'; c?: CanvasCard; z?: CanvasZone; ox: number; oy: number; sx: number; sy: number; vx: number; vy: number; moved?: boolean } | null = null
function startCardDrag(e: PointerEvent, c: CanvasCard) {
  if ((e.target as HTMLElement).closest('button') || (e.target as HTMLElement).closest('input') || (e.target as HTMLElement).closest('textarea')) return
  if (editing.value === c.id) return
  const w = toWorld(e.clientX, e.clientY)
  drag = { type: 'card', c, ox: w.x - c.x, oy: w.y - c.y, sx: 0, sy: 0, vx: 0, vy: 0 }
  selId.value = c.id
  try { (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId) } catch { /* 无活动指针时忽略 */ }
}
function startZoneDrag(e: PointerEvent, z: CanvasZone) {
  if ((e.target as HTMLElement).closest('button') || (e.target as HTMLElement).closest('input')) return
  const w = toWorld(e.clientX, e.clientY)
  drag = { type: 'zone', z, ox: w.x - z.x, oy: w.y - z.y, sx: 0, sy: 0, vx: 0, vy: 0 }
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
}
function onPointerDown(e: PointerEvent) {
  if (view.value !== 'board') return
  if ((e.target as HTMLElement).closest('.sc-card') || (e.target as HTMLElement).closest('.sc-zone') || (e.target as HTMLElement).closest('.sc-palette')) return
  drag = { type: 'pan', ox: 0, oy: 0, sx: e.clientX, sy: e.clientY, vx: viewT.value.x, vy: viewT.value.y }
  isPanning.value = true
  boardEl.value?.setPointerCapture(e.pointerId)
}
function onPointerMove(e: PointerEvent) {
  if (!drag) return
  if (drag.type === 'card' && drag.c) {
    const w = toWorld(e.clientX, e.clientY)
    drag.c.x = Math.round(w.x - drag.ox); drag.c.y = Math.round(w.y - drag.oy)
    drag.moved = true; drawEdgesSoon()
  } else if (drag.type === 'zone' && drag.z) {
    const w = toWorld(e.clientX, e.clientY)
    drag.z.x = Math.round(w.x - drag.ox); drag.z.y = Math.round(w.y - drag.oy)
    drag.moved = true
  } else if (drag.type === 'pan') {
    viewT.value.x = drag.vx + e.clientX - drag.sx
    viewT.value.y = drag.vy + e.clientY - drag.sy
  }
}
function onPointerUp() {
  if (drag && (drag.type === 'card' || drag.type === 'zone') && drag.moved) markDirty()
  drag = null; isPanning.value = false
}
function onWheel(e: WheelEvent) {
  if (view.value !== 'board') return
  e.preventDefault()
  zoomAt(e.clientX, e.clientY, e.deltaY < 0 ? 1.08 : 0.93)
}

/* ── 卡片操作 ── */
function kindLabel(c: CanvasCard): string {
  return c.kind === 'doc' ? (c.desc ? '角色' : '设定') : c.kind === 'graph' ? '关系图' : c.kind === 'quote' ? '摘录' : '便签'
}
function freeSpot(): { x: number; y: number } {
  const baseX = paletteOpen.value ? 300 : 80
  if (!cards.value.length) return { x: baseX, y: 70 }
  const maxY = Math.max(...cards.value.map(c => c.y + 190))
  return { x: baseX + (cards.value.length % 4) * 260, y: maxY + 40 }
}
function addDocCard(d: { id: number; title: string; desc?: string }) {
  if (cards.value.some(c => c.docId === d.id)) { toast.info('这张卡已经在画布上了'); return }
  const p = freeSpot()
  cards.value.push({ id: uid(), kind: 'doc', docId: d.id, title: d.title, desc: d.desc, x: p.x, y: p.y })
  paletteOpen.value = false // 收起抽屉让用户看到新卡
  markDirty(); toast.success(`「${d.title}」已上画布`)
}
function addFreeCard() {
  const p = freeSpot()
  cards.value.push({ id: uid(), kind: 'free', title: '新便签', text: '', x: p.x, y: p.y })
  paletteOpen.value = false
  markDirty()
}
function addQuoteFromSelection() {
  const sel = props.getSelection?.()
  if (!sel || !sel.text.trim()) { toast.info('先在正文里选中一段文字再摘录'); return }
  const p = freeSpot()
  cards.value.push({
    id: uid(), kind: 'quote', title: `摘录 · ${sel.chapterTitle}`,
    text: sel.text.slice(0, 600), src: `第 ${sel.chapterNo} 章 · ${sel.chapterTitle}`, x: p.x, y: p.y,
  })
  markDirty(); toast.success('已摘录到画布')
}
function removeCard(c: CanvasCard) {
  cards.value = cards.value.filter(x => x.id !== c.id)
  if (c.kind === 'doc') toast.info('已移出画布（文档本体还在）')
  markDirty(); drawEdgesSoon()
}
function togglePin(c: CanvasCard) {
  c.pinned = !c.pinned
  markDirty()
  toast.success(c.pinned ? '已钉选——右栏「设定」页签置顶显示' : '已取消钉选')
}
function startEdit(c: CanvasCard) { editing.value = c.id; nextTick(focusTa) }
function focusTa() {
  const ta = document.querySelector<HTMLTextAreaElement>('.sc-edit-textarea')
  ta?.focus()
}
watch(editing, (v) => { if (v === null) markDirty() })

/* ── 分区 ── */
function addZone() {
  const p = freeSpot()
  zones.value.push({ id: uid(), x: p.x - 10, y: p.y - 34, w: 560, h: 300, label: '新分区' })
  markDirty()
}
function editZoneLabel(z: CanvasZone) { editingZone.value = z.id; nextTick(() => { const i = document.querySelector<HTMLInputElement>('.sc-zone-input'); i?.focus(); i?.select() }) }
function delZone(z: CanvasZone) { zones.value = zones.value.filter(x => x.id !== z.id); markDirty() }
watch(editingZone, v => { if (v === null) markDirty() })
function zoneStyle(z: CanvasZone) { return { left: z.x + 'px', top: z.y + 'px', width: z.w + 'px', height: z.h + 'px' } }

/* ── 关系图引入（只读铺开；关系数据归关系图谱管） ── */
const graphEdges = ref<{ d: string; mx: number; my: number; label: string; color: string; dash?: string }[]>([])
async function importGraph() {
  importing.value = true
  try {
    const res = await memoryApi.getRelationships(String(props.projectId))
    const data = res.data?.data || res.data || {}
    const nodes: any[] = data.nodes || []
    const links: any[] = data.links || []
    const types: any[] = data.types || []
    if (!nodes.length) { toast.info('关系图谱里还没有角色，先去图谱里建几个'); return }
    // 节点 → 卡片（复用同名 doc 卡；缺则建 graph 卡）
    let added = 0
    for (const n of nodes) {
      const exist = cards.value.find(c => c.title === n.name && (c.kind === 'doc' || c.kind === 'graph'))
      if (exist) continue
      const doc = allDocs.value.find((d: any) => d.type === 'character' && d.title === n.name)
      const p = freeSpot()
      cards.value.push({
        id: uid(), kind: doc ? 'doc' : 'graph', docId: doc?.id, title: n.name,
        desc: doc ? descOf(doc) : (n.role || ''), x: p.x, y: p.y,
      })
      added++
    }
    // 卡片排布：首次引入时环形布局更好看
    if (added > 2) layoutCircle(nodes)
    ;(window as any).__relLinks = links
    ;(window as any).__relTypes = types
    await nextTick(); drawEdges(); markDirty()
    toast.success(`已引入 ${nodes.length} 个角色 · ${links.length} 条关系（只读，编辑去关系图谱）`)
  } catch { toast.error('关系图读取失败') } finally { importing.value = false }
}
function layoutCircle(nodes: any[]) {
  const names = new Set(nodes.map((n: any) => n.name))
  const mine = cards.value.filter(c => names.has(c.title) && (c.kind === 'doc' || c.kind === 'graph'))
  const R = 140 + mine.length * 26
  mine.forEach((c, i) => {
    const a = (i / mine.length) * Math.PI * 2 - Math.PI / 2
    c.x = Math.round(420 + R * Math.cos(a)); c.y = Math.round(330 + R * Math.sin(a))
  })
}
function drawEdges() {
  const links: any[] = (window as any).__relLinks || []
  const types: any[] = (window as any).__relTypes || []
  const byTitle = new Map(cards.value.filter(c => c.kind === 'doc' || c.kind === 'graph').map(c => [c.title, c]))
  const out: typeof graphEdges.value = []
  for (const l of links) {
    const a = byTitle.get(l.source); const b = byTitle.get(l.target)
    if (!a || !b) continue
    const ha = cardH(a.id) / 2, hb = cardH(b.id) / 2
    const x1 = a.x + 116, y1 = a.y + ha, x2 = b.x + 116, y2 = b.y + hb
    const mx = (x1 + x2) / 2, my = (y1 + y2) / 2
    const t = types.find((x: any) => x.key === l.type)
    out.push({
      d: `M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`,
      mx, my, label: l.label || t?.label || l.type,
      color: t?.color || '#9aa79c', dash: t?.dash?.length ? t.dash.join(' ') : undefined,
    })
  }
  graphEdges.value = out
}
function cardH(id: string): number {
  const el = worldEl.value?.querySelector(`[data-cid="${id}"]`)
  return el ? (el as HTMLElement).offsetHeight : 150
}
let edgeTimer: ReturnType<typeof setTimeout> | null = null
function drawEdgesSoon() {
  if (edgeTimer) clearTimeout(edgeTimer)
  edgeTimer = setTimeout(drawEdges, 60)
}

/* ── 伏笔时间线 ── */
const threads = ref<any[]>([])
const selThread = ref<any>(null)
async function switchTimeline() {
  view.value = 'timeline'
  try {
    const res = await threadApi.list(props.projectId)
    threads.value = (Array.isArray(res.data?.data) ? res.data.data : []).filter((t: any) => t.status === 'open' || t.resolved_chapter)
  } catch { threads.value = [] }
}
const threadRows = computed(() => {
  const n = Math.max(props.chapters.length, 1)
  const cur = props.currentChapter || props.chapters.length
  return threads.value.map((t, idx) => {
    const from = Math.max(0, (t.planted_chapter || 1) - 1)
    const to = t.status === 'resolved' ? Math.min(n - 1, Math.max(0, (t.resolved_chapter || 1) - 1)) : null
    const open = t.status !== 'resolved'
    const late = open && cur - (t.planted_chapter || 1) > 15
    return {
      ...t, idx, from, to,
      left: (from / n) * 100,
      width: ((to != null ? to - from + 1 : n - from) / n) * 100,
      state: t.status === 'resolved' ? 'done' : late ? 'late' : 'open',
      planted: t.planted_chapter, resolved: t.resolved_chapter,
    }
  })
})
const tlStats = computed(() => ({
  done: threadRows.value.filter(t => t.state === 'done').length,
  open: threadRows.value.filter(t => t.state === 'open').length,
  late: threadRows.value.filter(t => t.state === 'late').length,
}))
function tlStateLabel(t: any): string { return t.state === 'done' ? '已回收' : t.state === 'late' ? '超期未收' : '埋设中' }

/* ── 保存 ── */
function payload() { return { cards: cards.value, zones: zones.value, view: viewT.value } }
function markDirty() {
  saveState.value = 'dirty'
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => flush(), 900)
}
async function flush(manual = false) {
  if (saveState.value === 'saved' && !manual) return
  saveState.value = 'saving'
  try {
    await memoryApi.saveCanvas(props.projectId, payload())
    saveState.value = 'saved'
    if (manual) toast.success('画布已保存')
  } catch { saveState.value = 'dirty'; toast.error('画布保存失败') }
}
function onClose() {
  flush()
  window.dispatchEvent(new CustomEvent('xzj-canvas-closed'))
  emit('close')
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && editing.value === null && editingZone.value === null) { onClose() }
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') { e.preventDefault(); flush(true) }
}
function beforeUnload() { if (saveState.value !== 'saved') flush() }

/* ── 装载 ── */
onMounted(async () => {
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('keydown', onKey)
  window.addEventListener('beforeunload', beforeUnload)
  const board = boardEl.value
  board?.addEventListener('pointerdown', onPointerDown)
  board?.addEventListener('wheel', onWheel, { passive: false })
  try {
    const [cres, dres, gres] = await Promise.allSettled([
      memoryApi.getCanvas(props.projectId),
      documentApi.list(String(props.projectId)),
      memoryApi.getRelationships(String(props.projectId)),
    ])
    if (cres.status === 'fulfilled') {
      const d = cres.value.data?.data || cres.value.data
      if (d && Array.isArray(d.cards)) {
        cards.value = d.cards
        zones.value = Array.isArray(d.zones) ? d.zones : []
        if (d.view) viewT.value = d.view
      }
    }
    if (dres.status === 'fulfilled') allDocs.value = Array.isArray(dres.value.data?.data) ? dres.value.data.data : []
    if (gres.status === 'fulfilled') {
      (window as any).__relLinks = gres.value.data?.data?.links || []
      ;(window as any).__relTypes = gres.value.data?.data?.types || []
    }
    await nextTick(); drawEdges()
  } finally { loading.value = false }
})
onBeforeUnmount(() => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('beforeunload', beforeUnload)
  if (saveTimer) clearTimeout(saveTimer)
  if (edgeTimer) clearTimeout(edgeTimer)
  flush()
})
watch(() => props.projectId, () => { /* 父组件会重建本组件 */ })
</script>

<style scoped>
.sc-overlay{position:fixed;inset:0;z-index:1200;background:var(--bg-primary,#f5f5f7);display:flex;flex-direction:column}
/* 顶栏 */
.sc-topbar{height:52px;display:flex;align-items:center;gap:10px;padding:0 14px;background:var(--bg-secondary,#fffdf8);
  border-bottom:1px solid var(--border-color,#e3ddd0);flex-shrink:0}
.sc-brand{font-weight:800;letter-spacing:1px;font-size:14px}
.sc-seg{display:flex;background:rgba(0,0,0,.05);border-radius:10px;padding:3px;margin-left:8px}
.sc-seg button{border:none;background:none;font-size:12.5px;padding:5px 14px;border-radius:8px;color:var(--text-secondary,#6e6e73);cursor:pointer}
.sc-seg button.on{background:var(--bg-secondary,#fff);color:var(--text-primary,#1d1d1f);font-weight:700;box-shadow:0 1px 4px rgba(0,0,0,.1)}
.sc-tools{display:flex;gap:6px;margin-left:10px}
.sc-tbtn{border:1px solid var(--border-color,#e3ddd0);background:var(--bg-secondary,#fff);border-radius:8px;
  padding:5px 11px;font-size:12px;color:var(--text-secondary,#6e6e73);cursor:pointer;white-space:nowrap}
.sc-tbtn:hover{border-color:var(--accent,#6aa877);color:var(--accent-strong,#4d8a5c)}
.sc-tbtn:disabled{opacity:.5;cursor:default}
.sc-spacer{flex:1}
.sc-status{font-size:11.5px;color:var(--text-tertiary,#aeaeb2);cursor:pointer;padding:4px 10px;border-radius:99px}
.sc-status.dirty{color:#b3593f;background:rgba(179,89,63,.08)}
.sc-zoom{display:flex;align-items:center;gap:4px;font-size:11.5px;color:var(--text-tertiary,#aeaeb2)}
.sc-zpct{min-width:38px;text-align:center}
.sc-close{border:none;background:none;font-size:15px;color:var(--text-tertiary,#aeaeb2);cursor:pointer;padding:6px 10px}
.sc-close:hover{color:var(--text-primary,#1d1d1f)}
/* 白板 */
.sc-board{flex:1;position:relative;overflow:hidden;cursor:grab;
  background-image:radial-gradient(circle,var(--dot-color,#d8d4c8) 1px,transparent 1px);background-size:26px 26px}
.sc-board.panning{cursor:grabbing}
.sc-world{position:absolute;left:0;top:0;width:4000px;height:3000px;transform-origin:0 0;will-change:transform}
.sc-svg{position:absolute;left:0;top:0;width:100%;height:100%;overflow:visible;pointer-events:none}
.sc-edge{fill:none;stroke-width:1.6;opacity:.75;pointer-events:stroke;cursor:pointer}
.sc-elabel{font-size:11px}
/* 分区 */
.sc-zone{position:absolute;border:1.6px dashed #cfc8b6;border-radius:18px;background:rgba(255,255,255,.3);pointer-events:none}
.sc-zone-bar{position:absolute;top:-13px;left:14px;right:auto;display:flex;align-items:center;gap:6px;pointer-events:auto}
.sc-zone-label{background:#efe9db;padding:2px 12px;border-radius:99px;font-size:12px;color:#8a8272;letter-spacing:2px;cursor:move}
.sc-zone-input{font-size:12px;padding:2px 8px;border-radius:99px;border:1px solid var(--accent,#6aa877);outline:none;width:120px}
.sc-zone-del{border:none;background:rgba(0,0,0,.06);color:#8a8272;border-radius:50%;width:18px;height:18px;font-size:9px;cursor:pointer}
/* 卡片 */
.sc-card{position:absolute;width:232px;background:var(--paper,#fffdf8);border-radius:12px;padding:10px 12px 12px;
  box-shadow:0 2px 10px rgba(60,50,20,.08),0 0 0 1px rgba(0,0,0,.05);cursor:grab;user-select:none;touch-action:none}
.sc-card.sel{box-shadow:0 10px 28px rgba(60,50,20,.18),0 0 0 1.5px var(--accent,#6aa877)}
.sc-card.k-quote{background:#fdf8ec}
.sc-card.pinned::after{content:'';position:absolute;top:-6px;right:16px;width:14px;height:14px;border-radius:4px 4px 4px 0;
  background:var(--accent,#6aa877);transform:rotate(45deg);box-shadow:0 1px 3px rgba(0,0,0,.2)}
.sc-card-bar{display:flex;align-items:center;justify-content:space-between;margin-bottom:5px}
.sc-card-kind{font-size:10px;color:var(--text-tertiary,#aeaeb2);letter-spacing:1px}
.sc-card-acts{display:flex;gap:2px;opacity:0;transition:.15s}
.sc-card:hover .sc-card-acts,.sc-card.pinned .sc-card-acts{opacity:1}
.sc-act{border:none;background:none;font-size:12px;cursor:pointer;padding:1px 4px;border-radius:4px}
.sc-act.on{opacity:1}
.sc-card-title{font-weight:700;font-size:14px;line-height:1.4;cursor:text}
.sc-card-text{font-size:11.5px;color:var(--text-secondary,#6e6e73);line-height:1.65;margin-top:4px;white-space:pre-wrap;word-break:break-word;cursor:text}
.sc-card-text.quote{color:#8a6d5a}
.sc-card-src{font-size:10px;color:#a08a5a;margin-top:6px}
.sc-edit-input,.sc-edit-textarea{width:100%;border:1px solid var(--accent,#6aa877);border-radius:7px;padding:5px 8px;
  font-size:12px;font-family:inherit;outline:none;background:var(--bg-secondary,#fff);color:var(--text-primary,#1d1d1f);margin-top:3px}
/* 空状态 */
.sc-empty{position:absolute;left:50%;top:44%;transform:translate(-50%,-50%);text-align:center;color:var(--text-tertiary,#aeaeb2);
  pointer-events:none;font-size:13px;line-height:2}
.sc-empty-icon{font-size:34px;margin-bottom:8px;opacity:.6}
.sc-empty .sub{font-size:11.5px;opacity:.8}
/* 素材抽屉 */
.sc-palette{position:absolute;left:14px;top:14px;width:250px;max-height:calc(100% - 60px);background:var(--bg-secondary,#fffdf8);
  border-radius:14px;box-shadow:0 12px 36px rgba(60,50,20,.16);display:flex;flex-direction:column;overflow:hidden;z-index:30}
.sc-palette-head{display:flex;gap:6px;padding:10px}
.sc-palette-head input{flex:1;border:1px solid var(--border-color,#e3ddd0);border-radius:8px;padding:6px 10px;font-size:12px;outline:none}
.sc-palette-body{overflow-y:auto;padding:0 10px 10px}
.sc-pal-cap{font-size:11px;font-weight:700;color:var(--text-tertiary,#aeaeb2);letter-spacing:1px;margin:10px 2px 6px}
.sc-pal-item{border-radius:9px;padding:7px 10px;cursor:pointer;display:flex;flex-direction:column;gap:2px}
.sc-pal-item:hover{background:rgba(106,168,119,.1)}
.sc-pal-item b{font-size:12.5px}
.sc-pal-item small{font-size:10.5px;color:var(--text-tertiary,#aeaeb2);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.sc-pal-none{font-size:11px;color:var(--text-tertiary,#aeaeb2);padding:4px 2px}
.sc-hintbar{position:absolute;bottom:12px;left:50%;transform:translateX(-50%);font-size:11px;color:var(--text-tertiary,#aeaeb2);
  background:rgba(255,255,255,.72);padding:6px 16px;border-radius:99px;backdrop-filter:blur(6px);pointer-events:none}
/* 时间线 */
.sc-timeline{flex:1;position:relative;overflow:auto;background:var(--bg-primary,#f5f5f7)}
.tl-head{position:sticky;top:0;display:flex;background:var(--bg-primary,#f5f5f7);z-index:5;border-bottom:1px solid var(--border-color,#e3ddd0)}
.tl-ch{flex:1;text-align:center;font-size:11px;color:var(--text-secondary,#6e6e73);padding:8px 0 6px;min-width:64px;
  overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.tl-grid{position:absolute;left:0;right:0;top:34px;bottom:0;display:flex}
.tl-grid i{flex:1;border-left:1px dashed #e5e0d3;min-width:64px}
.tl-rows{position:relative;min-height:340px}
.tl-row{position:absolute;left:130px;right:20px;height:64px}
.tl-rname{position:absolute;left:-126px;top:16px;width:118px;text-align:right;font-size:11.5px;color:var(--text-secondary,#6e6e73);
  overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.tl-track{position:absolute;left:0;right:0;top:14px;height:34px}
.tl-arc{position:absolute;height:34px;border-radius:99px;top:0;display:flex;align-items:center;justify-content:center;
  overflow:hidden;cursor:pointer;transition:transform .15s}
.tl-arc:hover{transform:scaleY(1.15)}
.tl-arc.done{background:linear-gradient(90deg,#8fc59b,#6aa877)}
.tl-arc.open{background:linear-gradient(90deg,#d9bd6e,#c9a34e)}
.tl-arc.late{background:linear-gradient(90deg,#d4756a,#c0504d)}
.tl-arc span{font-size:10.5px;color:#fff;white-space:nowrap;padding:0 10px;text-shadow:0 1px 2px rgba(0,0,0,.25)}
.tl-dot{position:absolute;width:9px;height:9px;border-radius:50%;background:#fff;top:50%;transform:translateY(-50%);box-shadow:0 0 0 2px rgba(0,0,0,.12)}
.tl-dot.l{left:-4px}.tl-dot.r{right:-4px}
.tl-none{padding:60px;text-align:center;color:var(--text-tertiary,#aeaeb2);font-size:12.5px}
.tl-legend{position:sticky;bottom:0;display:flex;gap:16px;font-size:11px;color:var(--text-secondary,#6e6e73);
  background:rgba(255,255,255,.8);padding:8px 18px;border-top:1px solid var(--border-color,#e3ddd0);backdrop-filter:blur(6px)}
.tl-legend i{display:inline-block;width:10px;height:10px;border-radius:3px;margin-right:5px;vertical-align:-1px}
.tl-tag{font-size:10px;padding:1px 8px;border-radius:99px;margin-left:6px}
.tl-tag.done{background:rgba(106,168,119,.14);color:#4d8a5c}
.tl-tag.open{background:rgba(201,163,78,.16);color:#8a6a2a}
.tl-tag.late{background:rgba(192,80,77,.12);color:#c0504d}
/* 伏笔详情 */
.sc-detail{position:fixed;right:20px;bottom:54px;width:290px;background:var(--paper,#fffdf8);border-radius:14px;
  padding:14px 16px;box-shadow:0 12px 36px rgba(60,50,20,.2);z-index:40}
.sc-detail h5{font-size:13.5px;margin-bottom:6px}
.sc-detail p{font-size:11.5px;color:var(--text-secondary,#6e6e73);line-height:1.7}
.sc-detail .sub{margin-top:4px}
.sc-detail-x{position:absolute;top:8px;right:10px;border:none;background:none;color:var(--text-tertiary,#aeaeb2);cursor:pointer}
/* 深色 */
:global(.dark) .sc-board{background-image:radial-gradient(circle,#3a3a3e 1px,transparent 1px)}
:global(.dark) .sc-zone{border-color:#4a4a4e;background:rgba(255,255,255,.03)}
:global(.dark) .sc-zone-label{background:#333338;color:#aeaeb2}
@media (max-width:768px){ .sc-tools .sc-tbtn{padding:5px 8px;font-size:11px} .sc-hintbar{display:none} }
</style>
