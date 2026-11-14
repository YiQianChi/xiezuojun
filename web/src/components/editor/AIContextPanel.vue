<template>
  <div class="context-panel" :class="{ collapsed: !open }">
    <div
      class="context-panel-toggle"
      @click="$emit('update:open', !open)"
      :title="open ? '收起' : (ENABLE_AI ? '展开设定 / 灵感 / 自检' : '展开设定 / 灵感')"
    >
      <svg v-if="open" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="9 18 15 12 9 6"/></svg>
      <template v-else>
        <!-- 方案一·瘦身：收起态常驻竖排文字，让右栏工作台可被发现（手机端 CSS 隐藏，保留箭头） -->
        <span class="cpt-label"><BookOpen :size="15"/> {{ ENABLE_AI ? '设定 · 灵感 · 自检' : '设定 · 灵感' }}</span>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="15 18 9 12 15 6"/></svg>
      </template>
    </div>

    <div class="context-panel-inner" v-show="open">
      <div class="context-header">
        <div class="sheet-grabber" @click="$emit('update:open', false)" title="收起面板"></div>
        <h4>{{ TAB_LABELS[tab] }}</h4>
        <button class="sheet-close" @click="$emit('update:open', false)" title="关闭"><X :size="15"/></button>
      </div>

      <!-- 页签：设定 / 灵感 / 自检（右栏工作台 C 方案；旧「速查/AI 参考」已拆并） -->
      <div class="panel-tabs">
        <button :class="{ on: tab === 'set' }" @click="switchTab('set')"><BookOpen :size="15"/> 速查</button>
        <button :class="{ on: tab === 'insp' }" @click="switchTab('insp')"><Lightbulb :size="15"/> 灵感</button>
        <!-- AI 全线暂下线（2026-11-09 Boss 指令），自检恢复时删 v-if -->
        <button v-if="ENABLE_AI" :class="{ on: tab === 'chk' }" @click="switchTab('chk')"><ShieldCheck :size="15"/> 自检</button>
        <button :class="{ on: tab === 'ds' }" @click="switchTab('ds')" title="拆书报告：大纲/人物/关系/节奏/套路"><svg class="tab-ic" viewBox="0 0 19 19"><path d="M3 4c2-1 4.5-1 6.5.5C11.5 3 14 3 16 4v10.5c-2-1-4.5-1-6.5.5-2-1.5-4.5-1.5-6.5-.5z"/><path d="M9.5 4.5v10.5"/></svg> 拆书</button>
      </div>

      <div class="context-body">
        <!-- ═══════════ 设定页签：搜索 + 上一章结尾 + 角色 + 设定(喂AI勾选) + 伏笔 ═══════════ -->
        <template v-if="tab === 'set'">
          <div class="quick-search">
            <input v-model="quickQuery" placeholder="搜索角色 / 设定 / 伏笔…" />
            <button class="canvas-entry" title="打开设定画布：卡片自由摆放整理" @click="$emit('open-canvas')"><LayoutDashboard :size="15"/> 画布</button>
          </div>

          <div v-if="loading" class="context-empty">
            <div class="ctx-icon"><Hourglass :size="28"/></div>
            <div>加载中...</div>
          </div>

          <div v-else-if="characters.length === 0 && openThreads.length === 0" class="context-empty">
            <div class="ctx-icon"><Lightbulb :size="28"/></div>
            <div>还没有可翻的设定<br><span class="hint">在左栏「作品相关」里建角色卡和设定文档</span></div>
          </div>

          <template v-else>
            <!-- 画布钉选的卡片（置顶对照） -->
            <div v-if="!quickQuery && pinnedCards.length" class="quick-section">
              <div class="quick-cap"><Pin :size="15"/> 画布钉选<span class="cap-sub">画布里钉的卡，写这章对照</span></div>
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

            <!-- 伏笔（点条目就地编辑；原工具箱伏笔面板已并入这里） -->
            <div class="quick-section">
              <div class="quick-cap">伏笔 · 埋设中<span class="cap-sub">{{ quickQuery ? (filteredThreads.length ? filteredThreads.length + ' 命中' : '无命中') : openThreads.length }}</span><button class="th-add" title="直接记一条（埋设章=当前章）" @click="startThreadNew"><Plus :size="13"/> 记一条</button></div>
              <div v-if="threadEditingId === 0" class="th-edit">
                <select v-model="threadForm.type" class="th-edit-select">
                  <option v-for="(label, key) in THREAD_TYPE_LABELS" :key="key" :value="key">{{ label }}</option>
                </select>
                <input v-model="threadForm.title" class="th-edit-input" placeholder="标题（一句话说清这条伏笔）">
                <div class="th-edit-row">
                  <label>埋设章</label>
                  <input v-model.number="threadForm.planted_chapter" type="number" class="th-edit-num">
                  <label>预计收</label>
                  <input v-model.number="threadForm.planned_chapter" type="number" class="th-edit-num" placeholder="选填">
                </div>
                <div class="th-edit-actions">
                  <button class="th-edit-cancel" @click="threadEditingId = null">取消</button>
                  <button class="th-edit-save" @click="saveThreadEdit">记下</button>
                </div>
              </div>
              <template v-if="filteredThreads.length">
                <div v-for="t in filteredThreads" :key="t.id">
                  <div class="qc-thread" title="点开编辑" @click="startThreadEdit(t)">
                    <span class="th-type" :class="t.type">{{ THREAD_TYPE_LABELS[t.type] || t.type }}</span>
                    <span class="th-title">{{ t.title }}</span>
                    <span v-if="t.planted_chapter" class="th-ch">第{{ t.planted_chapter }}章埋</span>
                    <span v-if="t.planned_chapter" class="th-ch plan" :class="{ overdue: (currentChapter || 0) > t.planned_chapter }">计划第{{ t.planned_chapter }}章收</span>
                    <button v-if="t.doc_id" class="th-resolve th-loc" title="跳回埋设处并高亮" @click.stop="emit('thread-locate', t)"><Crosshair :size="14"/></button>
                    <button class="th-resolve" title="在当前章回收这条伏笔" @click.stop="resolveThread(t)"><Check :size="15"/> 回收</button>
                  </div>
                  <div v-if="threadEditingId === t.id" class="th-edit">
                    <select v-model="threadForm.type" class="th-edit-select">
                      <option v-for="(label, key) in THREAD_TYPE_LABELS" :key="key" :value="key">{{ label }}</option>
                    </select>
                    <input v-model="threadForm.title" class="th-edit-input" placeholder="标题">
                    <textarea v-model="threadForm.description" class="th-edit-input" placeholder="描述（可选）" rows="2"></textarea>
                    <div class="th-edit-row">
                      <label>埋设章</label>
                      <input v-model.number="threadForm.planted_chapter" type="number" class="th-edit-num">
                      <label>回收章</label>
                      <input v-model.number="threadForm.resolved_chapter" type="number" class="th-edit-num">
                      <select v-model="threadForm.status" class="th-edit-select">
                        <option value="open">未回收</option>
                        <option value="resolved">已回收</option>
                        <option value="abandoned">已放弃</option>
                      </select>
                    </div>
                    <div class="th-edit-row">
                      <label>预计回收章</label>
                      <input v-model.number="threadForm.planned_chapter" type="number" class="th-edit-num" style="width:auto;flex:1" placeholder="选填，不填按 15 章提醒">
                    </div>
                    <div class="th-edit-actions">
                      <button class="th-edit-del" @click="deleteThreadEdit">删除</button>
                      <button class="th-edit-cancel" @click="threadEditingId = null">取消</button>
                      <button class="th-edit-save" @click="saveThreadEdit">保存</button>
                    </div>
                  </div>
                </div>
              </template>
              <div v-else-if="!quickQuery" class="th-empty">
                <GitBranch :size="15"/> 还没有埋设中的伏笔<br><b>选中正文右键/长按「记伏笔」，或点上方「记一条」</b>随手埋
              </div>
              <!-- 已回收/已放弃（默认收起） -->
              <template v-if="!quickQuery && doneThreads.length">
                <div class="quick-cap th-done-cap" @click="showDoneThreads = !showDoneThreads">
                  已回收 {{ doneThreads.length }} {{ showDoneThreads ? '▴' : '▾' }}
                </div>
                <template v-if="showDoneThreads">
                  <div v-for="t in doneThreads" :key="t.id">
                    <div class="qc-thread done" title="点开编辑" @click="startThreadEdit(t)">
                      <span class="th-type" :class="t.type">{{ THREAD_TYPE_LABELS[t.type] || t.type }}</span>
                      <span class="th-title">{{ t.title }}</span>
                      <span v-if="t.resolved_chapter" class="th-ch">第{{ t.resolved_chapter }}章收</span>
                      <span v-else class="th-ch">已放弃</span>
                      <button class="th-resolve" title="误点了？一键恢复成埋设中" @click.stop="restoreThread(t)"><Undo2 :size="14"/> 恢复</button>
                    </div>
                    <div v-if="threadEditingId === t.id" class="th-edit">
                      <select v-model="threadForm.type" class="th-edit-select">
                        <option v-for="(label, key) in THREAD_TYPE_LABELS" :key="key" :value="key">{{ label }}</option>
                      </select>
                      <input v-model="threadForm.title" class="th-edit-input" placeholder="标题">
                      <textarea v-model="threadForm.description" class="th-edit-input" placeholder="描述（可选）" rows="2"></textarea>
                      <div class="th-edit-row">
                        <label>埋设章</label>
                        <input v-model.number="threadForm.planted_chapter" type="number" class="th-edit-num">
                        <label>回收章</label>
                        <input v-model.number="threadForm.resolved_chapter" type="number" class="th-edit-num">
                        <select v-model="threadForm.status" class="th-edit-select">
                          <option value="open">未回收</option>
                          <option value="resolved">已回收</option>
                          <option value="abandoned">已放弃</option>
                        </select>
                      </div>
                      <div class="th-edit-actions">
                        <button class="th-edit-del" @click="deleteThreadEdit">删除</button>
                        <button class="th-edit-cancel" @click="threadEditingId = null">取消</button>
                        <button class="th-edit-save" @click="saveThreadEdit">保存</button>
                      </div>
                    </div>
                  </div>
                </template>
              </template>
            </div>

            <div v-if="quickQuery && !filteredCharacters.length && !filteredThreads.length" class="context-empty">
              <div>没有匹配「{{ quickQuery }}」的内容</div>
            </div>
          </template>
        </template>

        <!-- ═══════════ 灵感页签：点子库 + AI 出走向 ═══════════ -->
        <template v-else-if="tab === 'insp'">
          <div v-if="inspList.length === 0 && !genOut" class="context-empty">
            <div class="ctx-icon"><Lightbulb :size="28"/></div>
            <div>灵感库还空着<br><span class="hint">想到什么写什么，底部输入框回车即存</span></div>
          </div>
          <div v-for="n in inspList" :key="n.id" class="insp-item">
            <div class="insp-text">{{ n.content }}</div>
            <div class="insp-meta">
              <span>{{ fmtTime(n.created_at) }}</span>
              <span class="insp-acts">
                <button title="插入到正文光标处" @click="emit('insp-insert', n.content)"><ArrowRightToLine :size="15"/> 写进本章</button>
                <button title="删除" @click="removeInsp(n.id)"><X :size="15"/></button>
              </span>
            </div>
          </div>
          <button v-if="ENABLE_AI" class="insp-gen" :disabled="genLoading" @click="genIdeas">
            <Sparkles v-if="!genLoading" :size="15"/> {{ genLoading ? 'AI 正在想…' : '卡文了？AI 出 3 个走向' }}
          </button>
          <div v-if="genOut" class="insp-item ai">
            <div class="insp-text">{{ genOut }}</div>
            <div class="insp-meta">
              <span>AI 生成</span>
              <span class="insp-acts">
                <button @click="saveGenAsInsp">存进灵感库</button>
                <button @click="genOut = ''"><X :size="15"/> 不要</button>
              </span>
            </div>
          </div>
        </template>

        <!-- ═══════════ 自检页签：AI 味指数 + 证据 + 改写建议 + 敏感词 ═══════════ -->
        <template v-else-if="tab === 'chk'">
          <button class="check-btn" :disabled="checking" @click="runCheck">
            <ShieldCheck :size="15"/> {{ checking ? '检查中…（约 1-2 分钟）' : '检查本章（AI 味 + 敏感词）' }}
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
              <div class="ctx-icon"><CircleCheckBig :size="28"/></div>
              <div>没查出明显的 AI 味问题</div>
            </div>
            <div v-for="(it, i) in result.items" :key="i" class="evi">
              <div class="et">{{ it.type }}<span class="lv" :class="it.severity">{{ SEVERITY_LABELS[it.severity] || it.severity }}</span></div>
              <div class="sent">{{ it.sentence }}</div>
              <button class="fix" @click="toggleFix(i)"><Pencil v-if="!fixOpen[i]" :size="15"/> {{ fixOpen[i] ? '▲ 收起建议' : '看改写建议' }}</button>
              <div v-if="fixOpen[i]" class="fixbox">{{ it.suggestion }}</div>
            </div>
          </template>
          <div v-else-if="!checking" class="context-empty">
            <div class="ctx-icon"><ShieldCheck :size="28"/></div>
            <div>检查 AI 味与敏感词<br><span class="hint">指数越低越像人写的</span></div>
          </div>
        </template>

        <!-- ═══════════ 拆书页签：五件套报告挂右栏随时查（只读参考，不写创作数据） ═══════════ -->
        <template v-else-if="tab === 'ds'">
          <div v-if="!dsBooks.length" class="context-empty">
            <div class="ctx-icon"><BookOpen :size="28"/></div>
            <div>还没拆过书<br><span class="hint">拆一本想学的书，报告挂在这里随时查</span></div>
            <button class="check-btn" @click="$emit('open-dismantle')">打开拆书库</button>
          </div>
          <template v-else>
            <div class="ds-books">
              <button v-for="b in dsBooks" :key="b.id" :class="{ on: dsSel === b.id }" :disabled="b.status !== 'done'" @click="pickBook(b.id)">
                {{ b.title }}{{ b.status === 'extracting' ? ' · ' + b.progress + '%' : b.status === 'failed' ? ' · 失败' : '' }}
              </button>
            </div>
            <template v-if="dsReport">
              <div class="ds-kinds">
                <button v-for="k in dsKinds" :key="k.k" :class="{ on: dsKind === k.k }" @click="dsKind = k.k">{{ k.t }}</button>
              </div>
              <pre v-if="dsKind !== 'anno'" class="ds-out">{{ dsReport.outputs[dsKind] || '（无内容）' }}</pre>
              <template v-else>
                <div v-if="!dsReport.annotations.length" class="context-empty">这一本没有章节标注</div>
                <div v-for="(a, i) in dsReport.annotations" :key="i" class="ds-anno">
                  <span class="k" :data-k="a.kind">{{ a.kind }}</span>
                  <b>第{{ a.chapter_idx }}章 · {{ a.title }}</b>
                  <span class="n">{{ a.note }}</span>
                </div>
              </template>
            </template>
          </template>
        </template>
      </div>

      <!-- 随手记：右栏底部常驻输入框，三个页签都在（Boss 2026-11：编辑器内就地记，不用悬浮笔/切页签） -->
      <div class="qn-bar">
        <textarea v-model="inspDraft" rows="4" placeholder="随手记一句…点我就变大，回车存进灵感库，Shift+回车换行" @keydown.enter.exact.prevent="addInsp"></textarea>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { documentApi, threadApi, stickyApi, aiApi, memoryApi, dismantleApi, type StoryThread } from '../../core/api'
import { streamChat, friendlyAIError } from '../../core/sse'
import { toast } from '../../composables/useToast'
import { BookOpen, Lightbulb, ShieldCheck, LayoutDashboard, Hourglass, Pin, Check, GitBranch, ArrowRightToLine, X, Sparkles, Pencil, CircleCheckBig, Crosshair, Plus, Undo2 } from 'lucide-vue-next'

interface CharCard { id: number; title: string; avatar: string; roleKey: string; roleLabel: string; desc: string; tags: string[] }
interface InspNote { id: number; content: string; created_at: string }
interface CheckItem { type: string; severity: string; sentence: string; suggestion: string }
interface CheckResult { score: number; level: string; summary: string; items: CheckItem[] }

const props = defineProps<{
  open: boolean
  projectId: number
  prevChapterId?: number | null
  currentChapter?: number
  getText?: () => string
  sensitiveTotal?: number
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  jump: [docId: number]
  'thread-locate': [t: StoryThread]
  'open-canvas': []
  'open-dismantle': []
  'insp-insert': [text: string]
}>()

const TAB_LABELS: Record<string, string> = { set: '速查', insp: '灵感', chk: '自检', ds: '拆书' }
const THREAD_TYPE_LABELS: Record<string, string> = {
  foreshadowing: '伏笔', hook: '钩子', promise: '承诺', suspense: '悬念',
}
const SEVERITY_LABELS: Record<string, string> = { high: '重', mid: '中', low: '轻' }

/* ─── 页签（localStorage 记忆） ─── */
// AI 全线暂下线（2026-11-09 Boss 指令：AI 只留代码不接入，恢复=改 true + 删各处 v-if）
const ENABLE_AI = false
const tab = ref<'set' | 'insp' | 'chk' | 'ds'>('set')
try {
  const saved = localStorage.getItem('xzj-wb-tab')
  if (saved === 'insp' || saved === 'set' || saved === 'ds') tab.value = saved
  else if (saved === 'chk' && ENABLE_AI) tab.value = saved
} catch { /* ignore */ }
if (tab.value === 'ds') loadDsBooks() // localStorage 记忆恢复时也要拉数据
function switchTab(t: 'set' | 'insp' | 'chk' | 'ds') {
  tab.value = t
  if (t === 'ds') loadDsBooks()
  try { localStorage.setItem('xzj-wb-tab', t) } catch { /* ignore */ }
}

/* ─── 拆书页签：书选一本 → 五件套/标注只读查看 ─── */
const dsBooks = ref<any[]>([])
const dsSel = ref(0)
const dsReport = ref<any>(null)
const dsKind = ref('outline')
const dsKinds = [
  { k: 'outline', t: '大纲' }, { k: 'chars', t: '人物' }, { k: 'net', t: '关系' },
  { k: 'pulse', t: '节奏' }, { k: 'trick', t: '套路' }, { k: 'anno', t: '标注' },
]
async function loadDsBooks() {
  if (!props.projectId) return
  try {
    const res = await dismantleApi.list(props.projectId)
    dsBooks.value = res.data?.data || []
    const cur = dsBooks.value.find((b: any) => b.id === dsSel.value)
    const first = dsBooks.value.find((b: any) => b.status === 'done')
    if (!cur || cur.status !== 'done') { if (first) pickBook(first.id); else dsReport.value = null }
  } catch { /* 静默 */ }
}
async function pickBook(id: number) {
  dsSel.value = id
  dsReport.value = null
  try {
    const res = await dismantleApi.get(id)
    dsReport.value = res.data?.data || null
  } catch { /* 静默 */ }
}

/* ─── 设定页签数据（原速查/AI 参考合并） ─── */
const loading = ref(false)
const characters = ref<CharCard[]>([])
const threads = ref<StoryThread[]>([])
const quickQuery = ref('')
const expanded = ref<Set<number>>(new Set())
const contentCache = ref<Record<number, string>>({})
const prevTail = ref('')

const ROLE_LABELS: Record<string, string> = { main: '主角', support: '配角', villain: '反派', extra: '群演' }

const filteredCharacters = computed(() => {
  const q = quickQuery.value.trim()
  if (!q) return characters.value
  return characters.value.filter(c => c.title.includes(q) || c.desc.includes(q) || c.tags.some(t => t.includes(q)))
})
const openThreads = computed(() => threads.value.filter(t => t.status === 'open'))
const doneThreads = computed(() => threads.value.filter(t => t.status !== 'open').slice(0, 20))
/* 点条目就地编辑（原工具箱伏笔面板能力并入） */
const threadEditingId = ref<number | null>(null)
const threadForm = ref<Partial<StoryThread>>({})
const showDoneThreads = ref(false)
function startThreadEdit(t: StoryThread) {
  threadEditingId.value = threadEditingId.value === t.id ? null : t.id!
  threadForm.value = { ...t }
}
function startThreadNew() {
  threadEditingId.value = 0 // 0 = 新建
  threadForm.value = { type: 'foreshadowing', title: '', status: 'open', planted_chapter: props.currentChapter || null }
}
async function saveThreadEdit() {
  const f = threadForm.value
  const id = threadEditingId.value
  if (id === null || !f.title?.trim()) return
  try {
    if (id === 0) {
      const res = await threadApi.create(props.projectId, { ...f, project_id: props.projectId })
      threads.value.unshift(res.data?.data ?? { ...f, id: Date.now() } as StoryThread)
    } else {
      await threadApi.update(id, f)
      threads.value = threads.value.map(x => x.id === id ? { ...x, ...f } as StoryThread : x)
    }
    threadEditingId.value = null
    toast.success('已保存')
  } catch {
    toast.error('保存失败')
  }
}
async function deleteThreadEdit() {
  const id = threadEditingId.value
  if (!id || !confirm('删除这条伏笔？')) return
  try {
    await threadApi.delete(id)
    threads.value = threads.value.filter(x => x.id !== id)
    threadEditingId.value = null
    toast.success('已删除')
  } catch {
    toast.error('删除失败')
  }
}
const filteredThreads = computed(() => {
  const q = quickQuery.value.trim()
  if (!q) return openThreads.value.slice(0, 5)
  return openThreads.value.filter(t => (t.title || '').includes(q) || (t.description || '').includes(q)).slice(0, 8)
})
async function loadSettings() {
  if (!props.projectId) return
  loading.value = true
  try {
    const res = await documentApi.list(String(props.projectId))
    const data = res.data?.data
    const allDocs = Array.isArray(data) ? data : []
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
    const text = docToText(raw)
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
// content 可能是 Tiptap HTML，先剥标签再取文本，否则 <p></p> 原样露出
function docToText(raw: string): string {
  return raw.replace(/<[^>]+>/g, '').replace(/&nbsp;/g, ' ')
}
async function loadPrevTail() {
  const id = props.prevChapterId
  if (!id) { prevTail.value = ''; return }
  try {
    const res = await documentApi.get(id)
    const raw: string = res.data?.data?.content || res.data?.data?.search_text || ''
    const text = docToText(raw).split('\n').map(l => l.replace(/^#{1,6}\s*/, '').trim()).filter(Boolean).join('\n')
    const tail = text.slice(-300)
    prevTail.value = tail ? '……' + tail : '（上一章为空）'
  } catch {
    prevTail.value = '上一章内容加载失败'
  }
}
watch(() => props.prevChapterId, () => { loadPrevTail(); expanded.value = new Set() }, { immediate: true })

async function restoreThread(t: StoryThread) {
  if (!t.id) return
  try {
    await threadApi.update(t.id, { status: 'open', resolved_chapter: null })
    threads.value = threads.value.map(x => x.id === t.id ? { ...x, status: 'open' as const, resolved_chapter: null } : x)
    toast.success(`已恢复埋设中：「${t.title}」`)
  } catch {
    toast.error('恢复失败，请重试')
  }
}

async function resolveThread(t: StoryThread) {
  if (!t.id) return
  const id = t.id
  try {
    await threadApi.update(t.id, { status: 'resolved', resolved_chapter: props.currentChapter || null })
    threads.value = threads.value.map(x => x.id === id ? { ...x, status: 'resolved' as const, resolved_chapter: props.currentChapter || null } : x)
    showDoneThreads.value = true // 回收即展开已回收区：误点当场能看到「恢复」
    toast.withAction(`已回收：「${t.title}」` + (props.currentChapter ? ` · 第${props.currentChapter}章` : ''), {
      label: '撤回',
      onClick: async () => {
        try {
          await threadApi.update(id, { status: 'open', resolved_chapter: null })
          threads.value = threads.value.map(x => x.id === id ? { ...x, status: 'open' as const, resolved_chapter: null } : x)
          toast.success(`已撤回：「${t.title}」`)
        } catch {
          toast.error('撤回失败，点伏笔条目可手动改回')
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
    toast.success('已记进灵感库')
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

/* 数据加载入口：必须在全部 ref/函数声明之后（immediate watch 同步执行，
   写到 refs 之前会在初始化竞态下 TDZ 崩掉整个编辑器——已踩两次：'B'、画布钉选 'Re'） */
watch(() => props.projectId, () => { loadSettings(); loadInspirations(); loadPinned() }, { immediate: true })
</script>

<style scoped>
/* ── 拆书页签 ── */
.tab-ic { width: 13px; height: 13px; stroke: currentColor; stroke-width: 1.6; fill: none; stroke-linecap: round; stroke-linejoin: round; vertical-align: -2px; }
.ds-books { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 8px; }
.ds-books button { border: 1px solid var(--ed-line, #e2d8c2); background: rgba(255,255,255,.5); color: var(--ed-ink-2, #8a8272); border-radius: 99px; padding: 3.5px 10px; font-size: 11px; cursor: pointer; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ds-books button.on { background: rgba(106,168,119,.14); border-color: #6aa877; color: #6aa877; font-weight: 700; }
.ds-books button:disabled { opacity: .45; cursor: default; }
.ds-kinds { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 8px; }
.ds-kinds button { border: none; background: rgba(0,0,0,.04); color: var(--ed-ink-2, #8a8272); border-radius: 7px; padding: 4px 11px; font-size: 11px; cursor: pointer; }
.ds-kinds button.on { background: rgba(106,168,119,.14); color: #6aa877; font-weight: 700; }
.ds-out { white-space: pre-wrap; word-break: break-word; font-family: inherit; font-size: 12px; line-height: 1.85; color: var(--ed-ink, #33302a); background: rgba(255,255,255,.5); border: 1px solid var(--ed-line, #e2d8c2); border-radius: 10px; padding: 10px 12px; margin: 0; }
.ds-anno { padding: 7px 0; border-bottom: 1px solid var(--ed-line-soft, #e8dfc8); font-size: 11.5px; display: flex; gap: 7px; align-items: baseline; flex-wrap: wrap; }
.ds-anno b { color: var(--ed-ink, #33302a); font-size: 11.5px; }
.ds-anno .n { color: var(--ed-ink-2, #8a8272); width: 100%; padding-left: 34px; }
.ds-anno .k { flex: none; font-size: 9.5px; border-radius: 99px; padding: 1.5px 7px; font-weight: 700; background: rgba(106,168,119,.12); color: #6aa877; }
.ds-anno .k[data-k="钩子"] { background: rgba(201,163,78,.15); color: #a8842e; }
.ds-anno .k[data-k="意象"] { background: rgba(109,89,122,.14); color: #6d597a; }

.context-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  bottom: 12px;
  width: 340px;
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
  width: 30px;
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
/* 方案一·瘦身：收起态=30px 竖排文字条（可发现性），手机端隐藏文字走箭头 */
.context-panel.collapsed .context-panel-toggle {
  width: 30px;
  min-width: 30px;
  writing-mode: vertical-rl;
  gap: 6px;
  font-size: 10.5px;
  letter-spacing: 1.5px;
}
.cpt-label { white-space: nowrap; }
/* 手机端收起态是水平小条：文字放不下，隐藏 label 回到纯箭头 */
@media (max-width: 768px) {
  .context-panel.collapsed .context-panel-toggle { writing-mode: horizontal-tb; width: 24px; min-width: 24px; }
  .cpt-label { display: none; }
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
  padding: 14px 16px 12px;
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
  font-size: 13px;
  color: var(--text-tertiary, #aeaeb2);
  letter-spacing: 0.5px;
  font-weight: 600;
}
/* 页签 */
.panel-tabs {
  display: flex;
  gap: 6px;
  padding: 10px 14px 8px;
}
.panel-tabs button {
  flex: 1;
  border: none;
  background: rgba(0, 0, 0, 0.045);
  color: var(--text-secondary, #86868b);
  font-size: 13px;
  padding: 8px 0;
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
  padding: 8px 16px 16px;
}
.context-empty {
  padding: 30px 12px;
  text-align: center;
  color: var(--text-tertiary, #aeaeb2);
  font-size: 13px;
  line-height: 1.6;
}
.context-empty .ctx-icon { font-size: 24px; margin-bottom: 8px; opacity: 0.6; }
.context-empty .hint { color: var(--text-tertiary, #aeaeb2); opacity: 0.8; font-size: 12px; }
/* ─── 设定页签 ─── */
.quick-search { padding: 2px 0 6px; display: flex; gap: 6px; }
.quick-search input {
  flex: 1;
  min-width: 0;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.03);
  font-size: 13px;
  padding: 8px 12px;
  color: var(--text-primary, #1d1d1f);
  outline: none;
  transition: all 0.15s;
}
.canvas-entry {
  border: 1px solid rgba(106, 168, 119, 0.45);
  border-radius: 8px;
  background: rgba(106, 168, 119, 0.1);
  color: var(--accent, #5a9668);
  font-size: 12.5px;
  font-weight: 700;
  padding: 0 12px;
  cursor: pointer;
  white-space: nowrap;
}
.canvas-entry:hover { background: rgba(106, 168, 119, 0.2); }
.qc-ava.pin { background: linear-gradient(160deg, #e8d89a, #c9a34e); }
.quick-search input:focus { border-color: var(--accent, #6aa877); background: #fff; }
.quick-search input::placeholder { color: var(--text-tertiary, #aeaeb2); }
.prev-tail { margin-top: 2px; font-size: 13px; line-height: 1.8; max-height: 150px; overflow-y: auto; }
.quick-section { margin-bottom: 14px; }
.quick-cap {
  display: flex;
  align-items: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary, #86868b);
  padding: 10px 2px 6px;
  letter-spacing: 1px;
}
.th-add { margin-left: auto; display: inline-flex; align-items: center; gap: 3px; font-size: 12px; font-weight: 500; letter-spacing: 0; color: var(--text-tertiary, #aeaeb2); background: none; border: 1px dashed var(--border, #d9d9de); border-radius: 6px; padding: 2px 8px; cursor: pointer; }
.th-add:hover { color: var(--accent, #6aa877); border-color: var(--accent, #6aa877); }
.th-ch.plan.overdue { color: #c0563e; font-weight: 600; }
.th-loc { padding: 2px 4px; }
.quick-cap .cap-sub {
  font-weight: 400;
  font-size: 12px;
  color: var(--text-tertiary, #aeaeb2);
  margin-left: 6px;
  letter-spacing: 0;
}
.qc-char { display: flex; gap: 12px; padding: 12px 2px; border-bottom: 1px dashed rgba(0, 0, 0, 0.06); }
.qc-char:last-child { border-bottom: none; }
.qc-ava {
  width: 40px; height: 40px;
  border-radius: 12px;
  flex-shrink: 0;
  background: linear-gradient(160deg, #8fc59b, #6aa877);
  color: #fff;
  font-size: 15px;
  display: flex; align-items: center; justify-content: center;
}
.qc-ava.villain { background: linear-gradient(160deg, #d8987a, #b3593f); }
.qc-main { flex: 1; min-width: 0; }
.qc-name {
  font-size: 14.5px;
  font-weight: 600;
  color: var(--text-primary, #1d1d1f);
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  line-height: 1.4;
}
.qc-role {
  font-size: 11.5px;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(106, 168, 119, 0.14);
  color: #4d8a5c;
  flex-shrink: 0;
}
.qc-role.villain { background: rgba(190, 90, 60, 0.12); color: #b3593f; }
.qc-desc { font-size: 13px; color: var(--text-secondary, #86868b); margin-top: 3px; line-height: 1.5; }
.qc-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 5px; }
.qc-tag {
  font-size: 11.5px;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.045);
  color: var(--text-tertiary, #86868b);
}
.qc-bio {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-primary, #1d1d1f);
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  padding: 8px 10px;
  margin-top: 7px;
  white-space: pre-wrap;
  word-break: break-word;
}
.qc-thread {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 6px;
  font-size: 14px;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.05);
  cursor: pointer;
}
.qc-thread:hover .th-title { color: var(--accent, #6aa877); }
.qc-thread.done .th-title { color: var(--text-tertiary, #aeaeb2); text-decoration: line-through; text-decoration-color: rgba(0,0,0,0.2); }
.th-done-cap { cursor: pointer; user-select: none; }
.th-edit {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  margin: 2px 0 6px;
  border-radius: 8px;
  background: var(--ed-panel-2, rgba(0, 0, 0, 0.03));
  border: 1px solid var(--ed-line, rgba(0, 0, 0, 0.07));
}
.th-edit-select, .th-edit-input, .th-edit-num {
  padding: 6px 8px;
  border: 1px solid var(--ed-line, rgba(0, 0, 0, 0.12));
  border-radius: 6px;
  font-size: 13px;
  background: var(--ed-paper, #fff);
  color: var(--ed-ink, inherit);
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
.th-edit-input { resize: none; }
.th-edit-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-secondary, #86868b);
}
.th-edit-row label { flex-shrink: 0; }
.th-edit-num { width: 64px; flex: none; }
.th-edit-actions { display: flex; gap: 6px; justify-content: flex-end; }
.th-edit-actions button {
  border: none;
  border-radius: 6px;
  font-size: 12.5px;
  padding: 5px 12px;
  cursor: pointer;
}
.th-edit-save { background: var(--accent, #6aa877); color: #fff; }
.th-edit-cancel { background: rgba(0, 0, 0, 0.06); color: var(--text-secondary, #86868b); }
.th-edit-del { background: transparent; color: #d05656; }
.qc-thread:last-child { border-bottom: none; }
.th-type {
  font-size: 12px;
  padding: 2px 8px;
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
.th-ch { font-size: 12px; color: var(--text-tertiary, #aeaeb2); flex-shrink: 0; }
.th-resolve {
  border: 1px solid rgba(0, 0, 0, 0.1);
  background: transparent;
  border-radius: 6px;
  font-size: 12px;
  padding: 4px 9px;
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
/* ─── 灵感页签 ─── */
/* ─── 随手记常驻条（三个页签共用，原灵感页签输入框迁来） ─── */
.qn-bar { padding: 12px 14px 14px; border-top: 1px solid rgba(0, 0, 0, 0.05); flex-shrink: 0; }
.qn-bar textarea {
  width: 100%;
  box-sizing: border-box;
  border: 1px dashed var(--accent, #81bb8e);
  border-radius: 12px;
  padding: 12px 14px;
  font-size: 15px;
  line-height: 1.6;
  background: var(--bg-primary, #fff);
  outline: none;
  resize: none;
  font-family: inherit;
  transition: min-height 0.18s ease;
}
/* PC 随手点：聚焦即撑到半屏高，写长也不挤；失焦收回底部条 */
.qn-bar:focus-within textarea { min-height: 42vh; }
.qn-bar textarea:focus { border-style: solid; }
.qn-bar textarea::placeholder { color: var(--text-tertiary, #aeaeb2); }
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
