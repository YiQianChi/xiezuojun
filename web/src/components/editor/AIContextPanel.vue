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
        <h4>{{ tab === 'quick' ? '速查' : 'AI 参考设定' }}</h4>
        <button class="sheet-close" @click="$emit('update:open', false)" title="关闭">✕</button>
      </div>

      <!-- 页签：速查 / AI 参考 -->
      <div class="panel-tabs">
        <button :class="{ on: tab === 'quick' }" @click="switchTab('quick')">速查</button>
        <button :class="{ on: tab === 'ai' }" @click="switchTab('ai')">AI 参考</button>
      </div>

      <div class="context-body">
        <!-- ═══ 速查页签 ═══ -->
        <template v-if="tab === 'quick'">
          <!-- 搜索：过滤角色/设定/伏笔 -->
          <div class="quick-search">
            <input v-model="quickQuery" placeholder="搜索角色 / 设定 / 伏笔…" />
          </div>

          <!-- 上一章结尾（接梗用，非搜索时置顶展示） -->
          <div v-if="!quickQuery && prevChapterId" class="quick-section">
            <div class="quick-cap">上一章结尾<span class="cap-sub">接梗用</span></div>
            <div class="qc-bio prev-tail">{{ prevTail || '加载中…' }}</div>
          </div>

          <div v-if="loading" class="context-empty">
            <div class="ctx-icon">⏳</div>
            <div>加载中...</div>
          </div>

          <div v-else-if="characters.length === 0 && settingDocs.length === 0 && openThreads.length === 0" class="context-empty">
            <div class="ctx-icon">💡</div>
            <div>还没有可速查的内容<br><span class="hint">在左栏「作品相关」里建角色卡和设定文档后，这里随手翻</span></div>
          </div>

          <template v-else>
            <!-- 已钉住 -->
            <div v-if="!quickQuery && pinnedCards.length" class="quick-section">
              <div class="quick-cap">📌 已钉住<span class="cap-sub">常驻 · 最多 3 张</span></div>
              <div v-for="c in pinnedCards" :key="'p' + c.id" class="qc-char open">
                <div class="qc-ava" :class="{ villain: c.roleKey === 'villain' }">{{ c.avatar }}</div>
                <div class="qc-main">
                  <div class="qc-name" @click="toggleExpand(c.id)">
                    {{ c.title }}
                    <span class="qc-role" :class="{ villain: c.roleKey === 'villain' }">{{ c.roleLabel }}</span>
                    <button class="pin-btn pinned" title="取消钉住" @click.stop="togglePin(c.id)">📌</button>
                  </div>
                  <div v-if="c.desc" class="qc-desc">{{ c.desc }}</div>
                  <div v-if="c.tags && c.tags.length" class="qc-tags">
                    <span v-for="t in c.tags.slice(0, 4)" :key="t" class="qc-tag">{{ t }}</span>
                  </div>
                  <div v-if="isExpanded(c.id)" class="qc-bio">{{ excerptOf(c.id) }}</div>
                </div>
              </div>
            </div>

            <!-- 角色速查 -->
            <div v-if="filteredCharacters.length" class="quick-section">
              <div class="quick-cap">角色<span class="cap-sub">{{ quickQuery ? filteredCharacters.length + ' 命中' : characters.length }}</span></div>
              <div v-for="c in filteredCharacters" :key="c.id" class="qc-char" :class="{ open: isExpanded(c.id) }">
                <div class="qc-ava" :class="{ villain: c.roleKey === 'villain' }">{{ c.avatar }}</div>
                <div class="qc-main">
                  <div class="qc-name" @click="toggleExpand(c.id)">
                    {{ c.title }}
                    <span class="qc-role" :class="{ villain: c.roleKey === 'villain' }">{{ c.roleLabel }}</span>
                    <button class="pin-btn" :class="{ pinned: isPinned(c.id) }" :title="isPinned(c.id) ? '取消钉住' : '钉住常驻'" @click.stop="togglePin(c.id)">📌</button>
                  </div>
                  <div v-if="c.desc" class="qc-desc">{{ c.desc }}</div>
                  <div v-if="isExpanded(c.id)" class="qc-bio">{{ excerptOf(c.id) }}</div>
                </div>
              </div>
            </div>

            <!-- 设定速查 -->
            <div v-if="filteredSettingDocs.length" class="quick-section">
              <div class="quick-cap">设定<span class="cap-sub">{{ quickQuery ? filteredSettingDocs.length + ' 命中' : settingDocs.length }}</span></div>
              <div v-for="d in filteredSettingDocs" :key="d.id" class="qc-doc" :class="{ open: isExpanded(d.id) }">
                <div class="qc-drow" @click="toggleExpand(d.id)">
                  <span class="qc-dicon">📄</span>
                  <span class="qc-dtitle">{{ d.title }}</span>
                  <span class="qc-dtype">{{ d.type }}</span>
                </div>
                <div v-if="isExpanded(d.id)" class="qc-bio">{{ excerptOf(d.id) }}</div>
              </div>
            </div>

            <!-- 伏笔速查（常驻：空了也显示引导，写正文右键可随手记） -->
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

            <!-- 搜索无结果 -->
            <div v-if="quickQuery && !filteredCharacters.length && !filteredSettingDocs.length && !filteredThreads.length" class="context-empty">
              <div>没有匹配「{{ quickQuery }}」的内容</div>
            </div>
          </template>
        </template>

        <!-- ═══ AI 参考页签（原功能原样） ═══ -->
        <template v-else>
          <div v-if="loading" class="context-empty">
            <div class="ctx-icon">⏳</div>
            <div>加载中...</div>
          </div>

          <div v-else-if="settingDocs.length === 0" class="context-empty">
            <div class="ctx-icon">💡</div>
            <div>暂无设定文档<br><span class="hint">在「作品相关」中创建大纲/人物/世界观后，这里可以选择哪些参与 AI</span></div>
          </div>

          <template v-else>
            <div v-for="group in groupedDocs" :key="group.type" class="setting-group">
              <div class="group-header">
                <span class="group-label">{{ group.label }}</span>
                <span class="group-count">{{ group.enabled }}/{{ group.docs.length }}</span>
              </div>
              <div
                v-for="doc in group.docs"
                :key="doc.id"
                class="setting-row"
                :class="{ disabled: !doc.ai_include }"
                @click="toggleDoc(doc)"
              >
                <div class="row-toggle" :class="{ on: doc.ai_include }">
                  <span class="toggle-dot"></span>
                </div>
                <span class="row-title">{{ doc.title }}</span>
                <span v-if="doc.used !== undefined" class="row-tokens" :class="{ unused: !doc.used }">
                  {{ doc.used ? doc.tokens + 't' : '未用' }}
                </span>
              </div>
            </div>

            <div v-if="usedSettings.length > 0" class="context-stats">
              本次参考 {{ usedCount }}/{{ usedSettings.length }} 张 · {{ totalTokens }} tokens
            </div>
          </template>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { documentApi, threadApi, type StoryThread } from '../../core/api'
import { toast } from '../../composables/useToast'

export interface UsedSetting {
  id: number
  title: string
  type: 'outline' | 'characters' | 'worldbuilding'
  used: boolean
  tokens: number
}

interface SettingDocItem {
  id: number
  title: string
  type: string
  ai_include: boolean | null
  used?: boolean
  tokens?: number
}

interface CharCard {
  id: number
  title: string
  avatar: string
  roleKey: string
  roleLabel: string
  desc: string
  tags: string[]
}

const props = defineProps<{
  open: boolean
  projectId: number
  usedSettings: UsedSetting[]
  prevChapterId?: number | null
  currentChapter?: number
}>()

defineEmits<{
  'update:open': [v: boolean]
  jump: [docId: number]
}>()

const loading = ref(false)
const settingDocs = ref<SettingDocItem[]>([])
const characters = ref<CharCard[]>([])
const threads = ref<StoryThread[]>([])

const THREAD_TYPE_LABELS: Record<string, string> = {
  foreshadowing: '伏笔', hook: '钩子', promise: '承诺', suspense: '悬念',
}

/* ─── 速查页签状态 ─── */
const tab = ref<'quick' | 'ai'>('quick')
const expanded = ref<Set<number>>(new Set())
const contentCache = ref<Record<number, string>>({})
const pinnedIds = ref<number[]>([])

/* 搜索 + 上一章结尾 */
const quickQuery = ref('')
const prevTail = ref('')

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

watch(() => props.prevChapterId, () => loadPrevTail(), { immediate: true })

const PIN_MAX = 3
const ROLE_LABELS: Record<string, string> = { main: '主角', support: '配角', villain: '反派', extra: '群演' }
const TYPE_LABELS: Record<string, string> = {
  outline: '大纲', characters: '人物', worldbuilding: '世界观',
  storyline: '故事主线', chapter_outline: '章节大纲', note: '笔记',
}

function pinKey() { return `xzj-quick-pin-${props.projectId}` }

function loadPins() {
  try {
    const raw = localStorage.getItem(pinKey())
    pinnedIds.value = raw ? (JSON.parse(raw) as number[]).slice(0, PIN_MAX) : []
  } catch { pinnedIds.value = [] }
}

function savePins() {
  try { localStorage.setItem(pinKey(), JSON.stringify(pinnedIds.value)) } catch { /* 私密模式忽略 */ }
}

function isPinned(id: number) { return pinnedIds.value.includes(id) }

function togglePin(id: number) {
  if (isPinned(id)) {
    pinnedIds.value = pinnedIds.value.filter(x => x !== id)
  } else {
    pinnedIds.value.push(id)
    if (pinnedIds.value.length > PIN_MAX) pinnedIds.value = pinnedIds.value.shift() !== undefined ? pinnedIds.value.slice(1) : pinnedIds.value
  }
  savePins()
}

const pinnedCards = computed(() => characters.value.filter(c => isPinned(c.id)))

function switchTab(t: 'quick' | 'ai') {
  tab.value = t
  try { localStorage.setItem('xzj-context-tab', t) } catch { /* ignore */ }
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
    // 小传/设定是 markdown 文本：剥标题井号，压掉连续空白
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

/* ─── AI 参考页签（原逻辑） ─── */
const groupedDocs = computed(() => {
  const groups: { type: string; label: string; docs: SettingDocItem[]; enabled: number }[] = []
  for (const doc of settingDocs.value) {
    let g = groups.find(g => g.type === doc.type)
    if (!g) {
      g = { type: doc.type, label: doc.type, docs: [], enabled: 0 }
      groups.push(g)
    }
    g.docs.push(doc)
    if (doc.ai_include !== false) g.enabled++
  }
  return groups
})

const usedCount = computed(() => props.usedSettings.filter(s => s.used).length)
const totalTokens = computed(() => props.usedSettings.reduce((sum, s) => sum + (s.tokens || 0), 0))

// 合并 usedSettings 到 settingDocs 显示
watch(() => props.usedSettings, (used) => {
  if (used.length === 0) return
  const map = new Map(used.map(s => [s.id, s]))
  settingDocs.value = settingDocs.value.map(d => {
    const u = map.get(d.id)
    return u ? { ...d, used: u.used, tokens: u.tokens } : d
  })
}, { deep: true })

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
      .map((d: any) => ({
        id: d.id, title: d.title,
        type: TYPE_LABELS[d.type] || '其他',
        ai_include: d.ai_include,
      }))
    // 角色卡速查：type=character，meta 里取 role/desc/tags/avatar
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
    loadPins()
    // 伏笔（同伏笔面板数据源，失败不打扰速查其它部分）
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

// ponytail: load on mount, not on first open — panel is collapsed by default
watch(() => props.projectId, () => loadSettings(), { immediate: true })

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

/* 速查里一键回收：当前章作为回收章 */
async function resolveThread(t: StoryThread) {
  if (!t.id) return
  try {
    await threadApi.update(t.id, { status: 'resolved', resolved_chapter: props.currentChapter || null })
    threads.value = threads.value.map(x => x.id === t.id ? { ...x, status: 'resolved' as const, resolved_chapter: props.currentChapter || null } : x)
    toast.success(`已回收：「${t.title}」` + (props.currentChapter ? ` · 第${props.currentChapter}章` : ''))
  } catch {
    toast.error('回收失败，请重试')
  }
}

// 恢复上次页签
try {
  const saved = localStorage.getItem('xzj-context-tab')
  if (saved === 'quick' || saved === 'ai') tab.value = saved
} catch { /* ignore */ }
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
  text-transform: uppercase;
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

/* ─── 速查样式 ─── */
.quick-search { padding: 2px 0 6px; }
.quick-search input {
  width: 100%;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.03);
  font-size: 12px;
  padding: 6px 10px;
  color: var(--text-primary, #1d1d1f);
  outline: none;
  transition: all 0.15s;
}
.quick-search input:focus {
  border-color: var(--accent, #6aa877);
  background: #fff;
}
.quick-search input::placeholder { color: var(--text-tertiary, #aeaeb2); }

.prev-tail {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.8;
  max-height: 150px;
  overflow-y: auto;
}

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
.th-resolve:hover {
  color: #4d8a5c;
  border-color: var(--accent, #6aa877);
  background: rgba(106, 168, 119, 0.1);
}
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

.qc-char {
  display: flex;
  gap: 9px;
  padding: 7px 2px;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.06);
}
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
.pin-btn {
  margin-left: auto;
  border: none;
  background: transparent;
  font-size: 12px;
  opacity: 0.35;
  cursor: pointer;
  padding: 0 2px;
  flex-shrink: 0;
}
.pin-btn:hover { opacity: 0.8; }
.pin-btn.pinned { opacity: 1; }
.qc-desc {
  font-size: 11.5px;
  color: var(--text-secondary, #86868b);
  margin-top: 3px;
  line-height: 1.5;
}
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
  cursor: pointer;
  font-size: 12.5px;
}
.qc-drow:hover { background: rgba(0, 0, 0, 0.04); }
.qc-dicon { font-size: 13px; flex-shrink: 0; }
.qc-dtitle {
  flex: 1; min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-primary, #1d1d1f);
}
.qc-dtype { font-size: 10px; color: var(--text-tertiary, #aeaeb2); flex-shrink: 0; }

/* ─── AI 参考页签（原样式） ─── */
.setting-group { margin-bottom: 12px; }
.group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 4px 6px;
}
.group-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary, #86868b);
}
.group-count {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  font-variant-numeric: tabular-nums;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.12s;
  margin-bottom: 2px;
}
.setting-row:hover { background: rgba(0, 0, 0, 0.04); }
.setting-row.disabled { opacity: 0.5; }

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

.row-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--text-primary, #1d1d1f);
}

.row-tokens {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.row-tokens.unused { color: var(--text-tertiary, #c7c7cc); }

.context-stats {
  font-size: 11px;
  color: var(--text-tertiary, #aeaeb2);
  padding: 8px 4px 4px;
  border-top: 1px dashed rgba(0, 0, 0, 0.05);
  margin-top: 4px;
}
</style>
