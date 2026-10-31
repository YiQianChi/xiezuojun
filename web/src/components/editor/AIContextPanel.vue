<template>
  <div class="context-panel" :class="{ collapsed: !open }">
    <div
      class="context-panel-toggle"
      @click="$emit('update:open', !open)"
      :title="open ? '收起' : '展开'"
    >
      <svg v-if="open" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="9 18 15 12 9 6"/></svg>
      <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="15 18 9 12 15 6"/></svg>
    </div>

    <div class="context-panel-inner" v-show="open">
      <div class="context-header">
        <div class="sheet-grabber" @click="$emit('update:open', false)" title="收起面板"></div>
        <h4>{{ TAB_LABELS[tab] }}</h4>
        <button class="sheet-close" @click="$emit('update:open', false)" title="关闭">✕</button>
      </div>

      <!-- 页签：设定 / 灵感 / 自检（右栏工作台 C 方案；旧「速查/AI 参考」已拆并） -->
      <div class="panel-tabs">
        <button :class="{ on: tab === 'set' }" @click="switchTab('set')">📖 设定</button>
        <button :class="{ on: tab === 'insp' }" @click="switchTab('insp')">💡 灵感</button>
        <button :class="{ on: tab === 'chk' }" @click="switchTab('chk')">🛡 自检</button>
      </div>

      <div class="context-body">
        <!-- ═══════════ 设定页签：搜索 + 上一章结尾 + 角色 + 设定(喂AI勾选) + 伏笔 ═══════════ -->
        <template v-if="tab === 'set'">
          <div class="quick-search">
            <input v-model="quickQuery" placeholder="搜索角色 / 设定 / 伏笔…" />
            <button class="canvas-entry" title="打开设定画布：卡片自由摆放整理" @click="$emit('open-canvas')">📐 画布</button>
          </div>

          <div v-if="loading" class="context-empty">
            <div class="ctx-icon">⏳</div>
            <div>加载中...</div>
          </div>

          <div v-else-if="characters.length === 0 && settingDocs.length === 0 && openThreads.length === 0" class="context-empty">
            <div class="ctx-icon">💡</div>
            <div>还没有可翻的设定<br><span class="hint">在左栏「作品相关」里建角色卡和设定文档</span></div>
          </div>

          <template v-else>
            <!-- 画布钉选的卡片（置顶对照） -->
            <div v-if="!quickQuery && pinnedCards.length" class="quick-section">
              <div class="quick-cap">📌 画布钉选<span class="cap-sub">画布里钉的卡，写这章对照</span></div>
              <div v-for="p in pinnedCards" :key="'pin' + p.id" class="qc-char">
                <div class="qc-ava pin">{{ p.kind === 'quote' ? '❝' : (p.title || '📌')[0] }}</div>
                <div class="qc-main">
                  <div class="qc-name" @click="jumpPinned(p)">{{ p.title }}</div>
                  <div class="qc-desc">{{ (p.text || p.desc || '').slice(0, 70) }}</div>
                </div>
              </div>
            </div>
            <!-- 上一章结尾（接梗用，非搜索时置顶） -->
            <div v-if="!quickQuery && prevChapterId" class="quick-section">
              <div class="quick-cap">上一章结尾<span class="cap-sub">接梗用</span></div>
              <div class="qc-bio prev-tail">{{ prevTail || '加载中…' }}</div>
            </div>

            <!-- 角色卡 -->
            <div v-if="filteredCharacters.length" class="quick-section">
              <div class="quick-cap">角色<span class="cap-sub">{{ characters.length }}</span></div>
              <div v-for="c in filteredCharacters" :key="c.id" class="qc-char">
                <div class="qc-ava" :class="{ villain: c.roleKey === 'villain' }">{{ c.avatar }}</div>
                <div class="qc-main">
                  <div class="qc-name" @click="toggleExpand(c.id)">
                    {{ c.title }}
                    <span class="qc-role" :class="{ villain: c.roleKey === 'villain' }">{{ c.roleLabel }}</span>
                  </div>
                  <div v-if="c.desc" class="qc-desc">{{ c.desc }}</div>
                  <div v-if="c.tags && c.tags.length" class="qc-tags"><span v-for="t in c.tags.slice(0, 4)" :key="t" class="qc-tag">{{ t }}</span></div>
                  <div v-if="isExpanded(c.id)" class="qc-bio">{{ excerptOf(c.id) }}</div>
                </div>
              </div>
            </div>

            <!-- 设定文档（喂 AI 勾选；原「AI 参考」并入此处） -->
            <div v-if="filteredSettingDocs.length" class="quick-section">
              <div class="quick-cap">设定<span class="cap-sub">点亮 = 喂给 AI 续写</span></div>
              <div v-for="d in filteredSettingDocs" :key="d.id" class="qc-doc">
                <div class="qc-drow">
                  <div class="row-toggle" :class="{ on: d.ai_include !== false }" :title="d.ai_include !== false ? '喂给 AI（点击关闭）' : '不喂 AI（点击开启）'" @click.stop="toggleDoc(d)"><span class="toggle-dot"></span></div>
                  <span class="qc-dicon">📄</span>
                  <span class="qc-dtitle" @click="toggleExpand(d.id)">{{ d.title }}</span>
                  <span class="qc-dtype">{{ d.type }}</span>
                </div>
                <div v-if="isExpanded(d.id)" class="qc-bio">{{ excerptOf(d.id) }}</div>
              </div>
              <div v-if="usedSettings.length > 0" class="context-stats">
                本次参考 {{ usedCount }}/{{ usedSettings.length }} 张 · {{ totalTokens }} tokens
              </div>
            </div>

            <!-- 伏笔 -->
            <div class="quick-section">
              <div class="quick-cap">伏笔 · 埋设中<span class="cap-sub">{{ quickQuery ? (filteredThreads.length ? filteredThreads.length + ' 命中' : '无命中') : openThreads.length }}</span></div>
              <template v-if="filteredThreads.length">
                <div v-for="t in filteredThreads" :key="t.id" class="qc-thread">
                  <span class="th-type" :class="t.type">{{ THREAD_TYPE_LABELS[t.type] || t.type }}</span>
                  <span class="th-title">{{ t.title }}</span>
                  <span v-if="t.planted_chapter" class="th-ch">第{{ t.planted_chapter }}章埋</span>
                  <button class="th-resolve" title="在当前章回收这条伏笔" @click="resolveThread(t)">✓ 回收</button>
                </div>
              </template>
              <div v-else-if="!quickQuery" class="th-empty">
                🧵 还没有埋设中的伏笔<br><b>选中正文 → 右键「记伏笔」</b>随手埋一条
              </div>
            </div>

            <div v-if="quickQuery && !filteredCharacters.length && !filteredSettingDocs.length && !filteredThreads.length" class="context-empty">
              <div>没有匹配「{{ quickQuery }}」的内容</div>
            </div>
          </template>
        </template>

        <!-- ═══════════ 灵感页签：点子库 + AI 出走向 ═══════════ -->
        <template v-else-if="tab === 'insp'">
          <div class="insp-add">
            <input v-model="inspDraft" placeholder="剧情点子，回车保存" @keydown.enter="addInsp" />
          </div>
          <div v-if="inspList.length === 0 && !genOut" class="context-empty">
            <div class="ctx-icon">💡</div>
            <div>灵感库还空着<br><span class="hint">想到什么写什么，卡文了让 AI 出走向</span></div>
          </div>
          <div v-for="n in inspList" :key="n.id" class="insp-item">
            <div class="insp-text">{{ n.content }}</div>
            <div class="insp-meta">
              <span>{{ fmtTime(n.created_at) }}</span>
              <span class="insp-acts">
                <button title="插入到正文光标处" @click="emit('insp-insert', n.content)">➜ 写进本章</button>
                <button title="删除" @click="removeInsp(n.id)">✕</button>
              </span>
            </div>
          </div>
          <button class="insp-gen" :disabled="genLoading" @click="genIdeas">
            {{ genLoading ? 'AI 正在想…' : '✦ 卡文了？AI 出 3 个走向' }}
          </button>
          <div v-if="genOut" class="insp-item ai">
            <div class="insp-text">{{ genOut }}</div>
            <div class="insp-meta">
              <span>AI 生成</span>
              <span class="insp-acts">
                <button @click="saveGenAsInsp">存进灵感库</button>
                <button @click="genOut = ''">✕ 不要</button>
              </span>
            </div>
          </div>
        </template>

        <!-- ═══════════ 自检页签：AI 味指数 + 证据 + 改写建议 + 敏感词 ═══════════ -->
        <template v-else-if="tab === 'chk'">
          <button class="check-btn" :disabled="checking" @click="runCheck">
            {{ checking ? '🛡 检查中…（约 1-2 分钟）' : '🛡 检查本章（AI 味 + 敏感词）' }}
          </button>

          <template v-if="result">
            <div class="score-ring">
              <div class="ring" :style="ringStyle"><i>{{ result.score }}</i></div>
              <div class="lab">
                <b>AI 味 {{ result.score }} · {{ result.level }}</b>
                <div v-if="result.summary" class="lab-sum">{{ result.summary }}</div>
                <div class="lab-sub">敏感词 {{ sensitiveTotal || 0 }} 处{{ sensitiveTotal ? '（状态栏 ◉ 查看）' : '' }}</div>
              </div>
            </div>
            <div v-if="result.items.length === 0" class="context-empty">
              <div class="ctx-icon">✅</div>
              <div>没查出明显的 AI 味问题</div>
            </div>
            <div v-for="(it, i) in result.items" :key="i" class="evi">
              <div class="et">{{ it.type }}<span class="lv" :class="it.severity">{{ SEVERITY_LABELS[it.severity] || it.severity }}</span></div>
              <div class="sent">{{ it.sentence }}</div>
              <button class="fix" @click="toggleFix(i)">{{ fixOpen[i] ? '▲ 收起建议' : '✎ 看改写建议' }}</button>
              <div v-if="fixOpen[i]" class="fixbox">{{ it.suggestion }}</div>
            </div>
          </template>
          <div v-else-if="!checking" class="context-empty">
            <div class="ctx-icon">🛡</div>
            <div>检查 AI 味与敏感词<br><span class="hint">指数越低越像人写的</span></div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { documentApi, threadApi, stickyApi, aiApi, memoryApi, type StoryThread } from '../../core/api'
import { streamChat, friendlyAIError } from '../../core/sse'
import { toast } from '../../composables/useToast'

export interface UsedSetting {
  id: number
  title: string
  tokens?: number
  used?: boolean
}

interface SettingDocItem { id: number; title: string; type: string; ai_include: boolean | null }
interface CharCard { id: number; title: string; avatar: string; roleKey: string; roleLabel: string; desc: string; tags: string[] }
interface InspNote { id: number; content: string; created_at: string }
interface CheckItem { type: string; severity: string; sentence: string; suggestion: string }
interface CheckResult { score: number; level: string; summary: string; items: CheckItem[] }

const props = defineProps<{
  open: boolean
  projectId: number
  usedSettings: UsedSetting[]
  prevChapterId?: number | null
  currentChapter?: number
  getText?: () => string
  sensitiveTotal?: number
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  jump: [docId: number]
  'open-canvas': []
  'insp-insert': [text: string]
}>()

const TAB_LABELS: Record<string, string> = { set: '📖 设定', insp: '💡 灵感', chk: '🛡 自检' }
const THREAD_TYPE_LABELS: Record<string, string> = {
  foreshadowing: '伏笔', hook: '钩子', promise: '承诺', suspense: '悬念',
}
const SEVERITY_LABELS: Record<string, string> = { high: '重', mid: '中', low: '轻' }

/* ─── 页签（localStorage 记忆） ─── */
const tab = ref<'set' | 'insp' | 'chk'>('set')
try {
  const saved = localStorage.getItem('xzj-wb-tab')
  if (saved === 'insp' || saved === 'chk' || saved === 'set') tab.value = saved
} catch { /* ignore */ }
function switchTab(t: 'set' | 'insp' | 'chk') {
  tab.value = t
  try { localStorage.setItem('xzj-wb-tab', t) } catch { /* ignore */ }
}

/* ─── 设定页签数据（原速查/AI 参考合并） ─── */
const loading = ref(false)
const settingDocs = ref<SettingDocItem[]>([])
const characters = ref<CharCard[]>([])
const threads = ref<StoryThread[]>([])
const quickQuery = ref('')
const expanded = ref<Set<number>>(new Set())
const contentCache = ref<Record<number, string>>({})
const prevTail = ref('')

const TYPE_LABELS: Record<string, string> = {
  outline: '大纲', characters: '人物', worldbuilding: '世界观',
  chapter_outline: '章节大纲', note: '笔记', setting: '设定',
}
const ROLE_LABELS: Record<string, string> = { main: '主角', support: '配角', villain: '反派', extra: '群演' }

const filteredCharacters = computed(() => {
  const q = quickQuery.value.trim()
  if (!q) return characters.value
  return characters.value.filter(c => c.title.includes(q) || c.desc.includes(q) || c.tags.some(t => t.includes(q)))
})
const filteredSettingDocs = computed(() => {
  const q = quickQuery.value.trim()
  if (!q) return settingDocs.value
  return settingDocs.value.filter(d => d.title.includes(q))
})
const openThreads = computed(() => threads.value.filter(t => t.status === 'open'))
const filteredThreads = computed(() => {
  const q = quickQuery.value.trim()
  if (!q) return openThreads.value.slice(0, 5)
  return openThreads.value.filter(t => (t.title || '').includes(q) || (t.description || '').includes(q)).slice(0, 8)
})
const usedCount = computed(() => props.usedSettings.filter(s => s.used).length)
const totalTokens = computed(() => props.usedSettings.reduce((sum, s) => sum + (s.tokens || 0), 0))

async function loadSettings() {
  if (!props.projectId) return
  loading.value = true
  try {
    const res = await documentApi.list(String(props.projectId))
    const data = res.data?.data
    const allDocs = Array.isArray(data) ? data : []
    const SETTING_TYPES = Object.keys(TYPE_LABELS)
    settingDocs.value = allDocs
      .filter((d: any) => SETTING_TYPES.includes(d.type))
      .map((d: any) => ({ id: d.id, title: d.title, type: TYPE_LABELS[d.type] || '其他', ai_include: d.ai_include }))
    // 角色卡：type=character，meta 里取 role/desc/tags/avatar
    characters.value = allDocs
      .filter((d: any) => d.type === 'character')
      .map((d: any) => {
        let meta: any = d.meta
        if (typeof meta === 'string') { try { meta = JSON.parse(meta) } catch { meta = {} } }
        meta = meta || {}
        const roleKey = String(meta.role || 'extra')
        return {
          id: d.id,
          title: d.title || '未命名',
          avatar: meta.avatar || (d.title || '人')[0],
          roleKey,
          roleLabel: ROLE_LABELS[roleKey] || '角色',
          desc: String(meta.desc || meta.identity || '').slice(0, 60),
          tags: Array.isArray(meta.tags) ? meta.tags.map(String) : [],
        }
      })
    contentCache.value = {}
    // 伏笔（同伏笔面板数据源，失败不打扰其它部分）
    try {
      const tres = await threadApi.list(props.projectId)
      const tdata = tres.data?.data
      threads.value = Array.isArray(tdata) ? tdata : []
    } catch { threads.value = [] }
  } catch {
    toast.error('加载设定列表失败')
  } finally {
    loading.value = false
  }
}

function isExpanded(id: number) { return expanded.value.has(id) }

function toggleExpand(id: number) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
  if (next.has(id)) ensureContent(id)
}

function excerptOf(id: number): string {
  const t = contentCache.value[id]
  if (t) return t.length > 320 ? t.slice(0, 320) + '…' : t
  return '加载中…'
}

async function ensureContent(id: number) {
  if (contentCache.value[id] !== undefined) return
  try {
    const res = await documentApi.get(id)
    const raw: string = res.data?.data?.content || res.data?.data?.search_text || ''
    const text = raw
      .split('\n')
      .map(l => l.replace(/^#{1,6}\s*/, '').trim())
      .filter(Boolean)
      .join('\n')
      .replace(/\n{2,}/g, '\n')
    contentCache.value = { ...contentCache.value, [id]: text || '（空）' }
  } catch {
    contentCache.value = { ...contentCache.value, [id]: '内容加载失败' }
  }
}

/* 上一章结尾：切章时拉上一章正文尾部 300 字 */
async function loadPrevTail() {
  const id = props.prevChapterId
  if (!id) { prevTail.value = ''; return }
  try {
    const res = await documentApi.get(id)
    const raw: string = res.data?.data?.content || res.data?.data?.search_text || ''
    const text = raw.split('\n').map(l => l.replace(/^#{1,6}\s*/, '').trim()).filter(Boolean).join('\n')
    const tail = text.slice(-300)
    prevTail.value = tail ? '……' + tail : '（上一章为空）'
  } catch {
    prevTail.value = '上一章内容加载失败'
  }
}
watch(() => props.prevChapterId, () => { loadPrevTail(); expanded.value = new Set() }, { immediate: true })

async function toggleDoc(doc: SettingDocItem) {
  const newVal = doc.ai_include === false ? true : false
  doc.ai_include = newVal
  try {
    await documentApi.update(doc.id, { ai_include: newVal })
  } catch {
    doc.ai_include = !newVal
    toast.error('更新失败')
  }
}

async function resolveThread(t: StoryThread) {
  if (!t.id) return
  const id = t.id
  try {
    await threadApi.update(t.id, { status: 'resolved', resolved_chapter: props.currentChapter || null })
    threads.value = threads.value.map(x => x.id === id ? { ...x, status: 'resolved' as const, resolved_chapter: props.currentChapter || null } : x)
    toast.withAction(`已回收：「${t.title}」` + (props.currentChapter ? ` · 第${props.currentChapter}章` : ''), {
      label: '撤回',
      onClick: async () => {
        try {
          await threadApi.update(id, { status: 'open', resolved_chapter: null })
          threads.value = threads.value.map(x => x.id === id ? { ...x, status: 'open' as const, resolved_chapter: null } : x)
          toast.success(`已撤回：「${t.title}」`)
        } catch {
          toast.error('撤回失败，请到工具箱-伏笔面板手动改回')
        }
      },
    })
  } catch {
    toast.error('回收失败，请重试')
  }
}

/* 正文右键记伏笔后联动刷新（Editor dispatch 'xzj-threads-changed'） */
function onThreadsChanged() {
  if (!props.projectId) return
  threadApi.list(props.projectId).then(res => {
    const tdata = res.data?.data
    threads.value = Array.isArray(tdata) ? tdata : []
  }).catch(() => {})
}
onMounted(() => window.addEventListener('xzj-threads-changed', onThreadsChanged))
onBeforeUnmount(() => window.removeEventListener('xzj-threads-changed', onThreadsChanged))

/* ─── 灵感页签（sticky type=insp） ─── */
const inspDraft = ref('')
const inspList = ref<InspNote[]>([])
const genLoading = ref(false)
const genOut = ref('')

async function loadInspirations() {
  if (!props.projectId) return
  try {
    const res = await stickyApi.list(props.projectId)
    const all = res.data?.data || []
    inspList.value = all.filter((n: any) => n.type === 'insp')
  } catch { inspList.value = [] }
}
async function addInsp() {
  const v = inspDraft.value.trim()
  if (!v || !props.projectId) return
  try {
    const res = await stickyApi.create({ project_id: props.projectId, type: 'insp', content: v })
    inspList.value.unshift(res.data?.data)
    inspDraft.value = ''
  } catch { toast.error('保存失败') }
}
async function removeInsp(id: number) {
  inspList.value = inspList.value.filter(n => n.id !== id)
  try { await stickyApi.remove(id) } catch { /* 列表已更新，下次进来自然对齐 */ }
}
function fmtTime(t: string) {
  const d = new Date(t)
  return isNaN(+d) ? '' : `${d.getMonth() + 1}-${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
async function genIdeas() {
  if (!props.projectId) return
  const tail = (props.getText?.() || '').replace(/<[^>]+>/g, '').trim().slice(-1200)
  genLoading.value = true
  genOut.value = ''
  let buf = ''
  try {
    await streamChat(
      { project_id: String(props.projectId), messages: [{ role: 'user', content: `我卡文了。下面是我正在写的章节结尾：\n${tail || '（还没写）'}\n\n结合这段剧情，给我 3 个接下来可以走的方向，每个方向一两句话，要具体可写（不要泛泛的"加强冲突"）。直接列 1. 2. 3.` }] },
      { onContent: (c: string) => { buf += c; genOut.value = buf } },
    )
  } catch (e: any) {
    toast.error(friendlyAIError(e))
  } finally {
    genLoading.value = false
    if (!genOut.value.trim()) genOut.value = ''
  }
}
async function saveGenAsInsp() {
  if (!genOut.value.trim() || !props.projectId) return
  try {
    const res = await stickyApi.create({ project_id: props.projectId, type: 'insp', content: genOut.value.trim() })
    inspList.value.unshift(res.data?.data)
    genOut.value = ''
    toast.success('已存进灵感库')
  } catch { toast.error('保存失败') }
}

/* ─── 自检页签 ─── */
const checking = ref(false)
const result = ref<CheckResult | null>(null)
const fixOpen = reactive<Record<number, boolean>>({})
const ringStyle = computed(() => {
  const s = result.value?.score ?? 0
  const color = s >= 60 ? '#c0504d' : s >= 30 ? '#d97706' : '#6aa877'
  return { background: `conic-gradient(${color} 0 ${s}%, #eee3cc ${s}% 100%)` }
})
function toggleFix(i: number) { fixOpen[i] = !fixOpen[i] }
async function runCheck() {
  const text = (props.getText?.() || '').replace(/<[^>]+>/g, '').trim()
  if (!text) { toast.error('本章还没写内容'); return }
  checking.value = true
  result.value = null
  Object.keys(fixOpen).forEach(k => delete fixOpen[Number(k)])
  try {
    const res = await aiApi.selfcheck(text)
    result.value = res.data?.data
  } catch (e: any) {
    toast.error(e?.response?.data?.error || friendlyAIError(e))
  } finally {
    checking.value = false
  }
}

/* 数据加载入口：放在所有声明之后（immediate watch 同步执行，
   若写在灵感 refs 之前会在初始化竞态下触发 TDZ——踩过：Cannot access 'B' before initialization） */
watch(() => props.projectId, () => { loadSettings(); loadInspirations(); loadPinned() }, { immediate: true })

/* ─── 画布钉选：读取画布布局里 pinned 的卡片，置顶展示 ─── */
const pinnedCards = ref<{ id: string; kind: string; title: string; text?: string; desc?: string; docId?: number }[]>([])
async function loadPinned() {
  if (!props.projectId) { pinnedCards.value = []; return }
  try {
    const res = await memoryApi.getCanvas(props.projectId)
    const d = res.data?.data || res.data
    pinnedCards.value = (Array.isArray(d?.cards) ? d.cards : []).filter((c: any) => c.pinned)
  } catch { pinnedCards.value = [] }
}
function jumpPinned(p: { docId?: number }) {
  if (p.docId) emit('jump', p.docId)
}
// 画布关闭回来自动刷新钉选（Editor 上 canvasOpen 变化会重挂/触发不了这里，用窗口事件兜底）
function onCanvasClosed() { loadPinned() }
onMounted(() => window.addEventListener('xzj-canvas-closed', onCanvasClosed))
onBeforeUnmount(() => window.removeEventListener('xzj-canvas-closed', onCanvasClosed))
</script>

<style scoped>
.context-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  bottom: 12px;
  width: 280px;
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06), 0 0 0 1px rgba(0, 0, 0, 0.04);
  z-index: 300;
  display: flex;
  flex-direction: row;
  overflow: hidden;
  transition: width 0.2s ease;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}
.context-panel.collapsed {
  width: 24px;
  background: transparent;
  box-shadow: none;
}
.context-panel-toggle {
  width: 24px;
  min-width: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-secondary, #86868b);
  background: var(--bg-tertiary, #f0f0f0);
  border-radius: 6px 0 0 6px;
  transition: background 0.15s;
}
.context-panel.collapsed .context-panel-toggle {
  background: var(--bg-secondary, #ffffff);
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1), 0 0 0 1px rgba(0, 0, 0, 0.05);
}
.context-panel-toggle:hover { background: rgba(0, 0, 0, 0.08); color: var(--text-primary, #1d1d1f); }
.context-panel-inner {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.context-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 10px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}
/* ponytail-mobile: 底部抽屉抓手+关闭钮——桌面端隐藏，窄屏显示 */
.sheet-grabber { display: none; }
.sheet-close { display: none; border: none; background: transparent; color: var(--text-tertiary, #aeaeb2); font-size: 14px; padding: 4px 6px; cursor: pointer; }
@media (max-width: 768px) {
  .sheet-grabber {
    display: block;
    width: 36px;
    height: 4px;
    border-radius: 2px;
    background: rgba(0, 0, 0, 0.18);
    cursor: pointer;
  }
  .sheet-close { display: block; }
}
.context-header h4 {
  margin: 0;
  font-size: 12px;
  color: var(--text-tertiary, #aeaeb2);
  letter-spacing: 0.5px;
  font-weight: 600;
}
/* 页签 */
.panel-tabs {
  display: flex;
  gap: 4px;
  padding: 8px 12px 6px;
}
.panel-tabs button {
  flex: 1;
  border: none;
  background: rgba(0, 0, 0, 0.045);
  color: var(--text-secondary, #86868b);
  font-size: 12px;
  padding: 6px 0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}
.panel-tabs button.on {
  background: var(--accent-light, rgba(106, 168, 119, 0.15));
  color: var(--accent, #6aa877);
  font-weight: 600;
}
.context-body {
  flex: 1;
  overflow-y: auto;
  padding: 4px 12px 12px;
}
.context-empty {
  padding: 30px 12px;
  text-align: center;
  color: var(--text-tertiary, #aeaeb2);
  font-size: 12px;
  line-height: 1.6;
}
.context-empty .ctx-icon { font-size: 24px; margin-bottom: 8px; opacity: 0.6; }
.context-empty .hint { color: var(--text-tertiary, #aeaeb2); opacity: 0.8; font-size: 11px; }
/* ─── 设定页签 ─── */
.quick-search { padding: 2px 0 6px; display: flex; gap: 6px; }
.quick-search input {
  flex: 1;
  min-width: 0;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.03);
  font-size: 12px;
  padding: 6px 10px;
  color: var(--text-primary, #1d1d1f);
  outline: none;
  transition: all 0.15s;
}
.canvas-entry {
  border: 1px solid rgba(106, 168, 119, 0.45);
  border-radius: 8px;
  background: rgba(106, 168, 119, 0.1);
  color: var(--accent, #5a9668);
  font-size: 11.5px;
  font-weight: 700;
  padding: 0 10px;
  cursor: pointer;
  white-space: nowrap;
}
.canvas-entry:hover { background: rgba(106, 168, 119, 0.2); }
.qc-ava.pin { background: linear-gradient(160deg, #e8d89a, #c9a34e); }
.quick-search input:focus { border-color: var(--accent, #6aa877); background: #fff; }
.quick-search input::placeholder { color: var(--text-tertiary, #aeaeb2); }
.prev-tail { margin-top: 2px; font-size: 12px; line-height: 1.8; max-height: 150px; overflow-y: auto; }
.quick-section { margin-bottom: 10px; }
.quick-cap {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary, #86868b);
  padding: 6px 2px 4px;
  letter-spacing: 1px;
}
.quick-cap .cap-sub {
  font-weight: 400;
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  margin-left: 6px;
  letter-spacing: 0;
}
.qc-char { display: flex; gap: 9px; padding: 7px 2px; border-bottom: 1px dashed rgba(0, 0, 0, 0.06); }
.qc-char:last-child { border-bottom: none; }
.qc-ava {
  width: 32px; height: 32px;
  border-radius: 9px;
  flex-shrink: 0;
  background: linear-gradient(160deg, #8fc59b, #6aa877);
  color: #fff;
  font-size: 13px;
  display: flex; align-items: center; justify-content: center;
}
.qc-ava.villain { background: linear-gradient(160deg, #d8987a, #b3593f); }
.qc-main { flex: 1; min-width: 0; }
.qc-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary, #1d1d1f);
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  line-height: 1.4;
}
.qc-role {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(106, 168, 119, 0.14);
  color: #4d8a5c;
  flex-shrink: 0;
}
.qc-role.villain { background: rgba(190, 90, 60, 0.12); color: #b3593f; }
.qc-desc { font-size: 11.5px; color: var(--text-secondary, #86868b); margin-top: 3px; line-height: 1.5; }
.qc-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 5px; }
.qc-tag {
  font-size: 10px;
  padding: 1px 7px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.045);
  color: var(--text-tertiary, #86868b);
}
.qc-bio {
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-primary, #1d1d1f);
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  padding: 8px 10px;
  margin-top: 7px;
  white-space: pre-wrap;
  word-break: break-word;
}
.qc-doc { padding: 2px; }
.qc-drow {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 4px;
  border-radius: 8px;
  font-size: 12.5px;
}
.qc-drow:hover { background: rgba(0, 0, 0, 0.04); }
.qc-dicon { font-size: 13px; flex-shrink: 0; }
.qc-dtitle {
  flex: 1; min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
}
.qc-dtype { font-size: 10px; color: var(--text-tertiary, #aeaeb2); flex-shrink: 0; }
.row-toggle {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1.5px solid var(--text-tertiary, #c7c7cc);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
  cursor: pointer;
}
.row-toggle.on {
  border-color: var(--accent, #6aa877);
  background: var(--accent, #6aa877);
}
.toggle-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: white;
  opacity: 0;
  transition: opacity 0.15s;
}
.row-toggle.on .toggle-dot { opacity: 1; }
.qc-thread {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 6px 4px;
  font-size: 12px;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.05);
}
.qc-thread:last-child { border-bottom: none; }
.th-type {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(175, 140, 60, 0.13);
  color: #8a6a2a;
  flex-shrink: 0;
}
.th-type.hook { background: rgba(106, 168, 119, 0.13); color: #4d8a5c; }
.th-type.promise { background: rgba(90, 130, 180, 0.12); color: #4a6f96; }
.th-type.suspense { background: rgba(150, 100, 170, 0.12); color: #7a5490; }
.th-title {
  flex: 1; min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-primary, #1d1d1f);
}
.th-ch { font-size: 10px; color: var(--text-tertiary, #aeaeb2); flex-shrink: 0; }
.th-resolve {
  border: 1px solid rgba(0, 0, 0, 0.1);
  background: transparent;
  border-radius: 6px;
  font-size: 10px;
  padding: 2px 7px;
  color: var(--text-tertiary, #86868b);
  cursor: pointer;
  flex-shrink: 0;
}
.th-resolve:hover { color: #4d8a5c; border-color: var(--accent, #6aa877); background: rgba(106, 168, 119, 0.1); }
.th-empty {
  font-size: 12px;
  color: var(--text-secondary, #86868b);
  background: rgba(0, 0, 0, 0.025);
  border: 1px dashed rgba(0, 0, 0, 0.12);
  border-radius: 10px;
  padding: 12px;
  text-align: center;
  line-height: 1.7;
}
.th-empty b { color: #4d8a5c; font-weight: 600; }
.context-stats {
  font-size: 11px;
  color: var(--text-tertiary, #aeaeb2);
  padding: 8px 4px 4px;
  border-top: 1px dashed rgba(0, 0, 0, 0.05);
  margin-top: 4px;
}
/* ─── 灵感页签 ─── */
.insp-add { padding: 2px 0 8px; }
.insp-add input {
  width: 100%;
  border: 1px dashed var(--accent, #81bb8e);
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 12px;
  background: var(--bg-primary, #fff);
  outline: none;
}
.insp-add input:focus { border-style: solid; }
.insp-add input::placeholder { color: var(--text-tertiary, #aeaeb2); }
.insp-item {
  border-left: 3px solid var(--accent, #81bb8e);
  background: rgba(129, 187, 142, 0.07);
  border-radius: 0 8px 8px 0;
  padding: 8px 10px;
  margin-bottom: 7px;
}
.insp-item.ai { border-left-color: #7c5cc4; background: rgba(124, 92, 196, 0.06); }
.insp-text { font-size: 12px; line-height: 1.7; color: var(--text-primary, #1d1d1f); white-space: pre-wrap; word-break: break-word; }
.insp-meta { display: flex; align-items: center; margin-top: 5px; font-size: 10px; color: var(--text-tertiary, #86868b); }
.insp-acts { margin-left: auto; display: flex; gap: 8px; }
.insp-acts button { border: none; background: none; font-size: 10.5px; color: #5a9668; font-weight: 700; cursor: pointer; padding: 0; }
.insp-acts button:hover { text-decoration: underline; }
.insp-gen {
  width: 100%;
  border: 1px dashed var(--accent, #81bb8e);
  background: rgba(129, 187, 142, 0.1);
  color: #5a9668;
  border-radius: 9px;
  padding: 8px 0;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
}
.insp-gen:disabled { opacity: 0.55; cursor: default; }
/* ─── 自检页签 ─── */
.check-btn {
  width: 100%;
  border: none;
  border-radius: 10px;
  padding: 10px 0;
  font-size: 12.5px;
  font-weight: 800;
  color: #fff;
  background: linear-gradient(135deg, #8fc59b, #6aa877);
  cursor: pointer;
}
.check-btn:disabled { opacity: 0.6; cursor: default; }
.score-ring {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(129, 187, 142, 0.08);
  border-radius: 12px;
  padding: 10px 12px;
  margin: 10px 0;
}
.ring { width: 56px; height: 56px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.ring i {
  width: 42px; height: 42px; border-radius: 50%;
  background: var(--bg-secondary, #fff);
  font-style: normal;
  font-size: 15px;
  font-weight: 800;
  color: var(--text-primary, #1d1d1f);
  display: flex; align-items: center; justify-content: center;
}
.score-ring .lab { font-size: 11px; color: var(--text-secondary, #5b5b60); line-height: 1.7; }
.score-ring .lab b { font-size: 12.5px; color: var(--text-primary, #1d1d1f); }
.lab-sum { margin-top: 2px; }
.lab-sub { color: var(--text-tertiary, #86868b); }
.evi { border: 1px solid rgba(0, 0, 0, 0.08); border-radius: 10px; padding: 8px 10px; font-size: 11.5px; margin-bottom: 8px; }
.et { font-weight: 700; display: flex; align-items: center; gap: 6px; color: var(--text-primary, #1d1d1f); }
.lv { font-size: 9px; border-radius: 4px; padding: 1px 6px; font-weight: 800; }
.lv.high { background: #fde8e8; color: #c0504d; }
.lv.mid { background: #fef3c0; color: #a07816; }
.lv.low { background: #e8f0fe; color: #3b6fd4; }
.sent {
  color: #8a6d5a; margin: 5px 0; line-height: 1.7; font-size: 11px;
  background: rgba(0, 0, 0, 0.025); border-radius: 6px; padding: 6px 9px;
}
.fix { border: none; background: none; font-size: 10.5px; color: #5a9668; font-weight: 700; cursor: pointer; padding: 0; }
.fixbox { margin-top: 5px; background: rgba(129, 187, 142, 0.08); border-radius: 8px; padding: 7px 10px; color: #4d6b55; line-height: 1.7; font-size: 11px; }
</style>
