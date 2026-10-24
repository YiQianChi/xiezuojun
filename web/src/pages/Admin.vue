<template>
  <div class="admin-page">
    <header class="admin-header">
      <h1>写作菌 · 后台管理</h1>
      <nav class="admin-tabs">
        <button v-for="t in TABS" :key="t.id" class="admin-tab" :class="{ active: tab === t.id }" @click="tab = t.id">
          {{ t.icon }} {{ t.label }}
        </button>
      </nav>
      <button class="btn-exit" @click="router.push('/bookshelf')">← 回书架</button>
    </header>

    <main class="admin-main">
      <!-- 看板 -->
      <template v-if="tab === 'dashboard'">
        <div v-if="!isAdmin" class="no-perm">⛔ 无管理员权限（{{ userEmail }}）</div>
        <template v-else>
          <div class="stat-cards">
            <div class="stat-card"><div class="sc-num">{{ ov.user_total }}</div><div class="sc-label">注册用户</div></div>
            <div class="stat-card"><div class="sc-num">{{ ov.active_today }}</div><div class="sc-label">今日活跃（有写作）</div></div>
            <div class="stat-card"><div class="sc-num">{{ ov.new_users_7d }}</div><div class="sc-label">7 日新增</div></div>
            <div class="stat-card"><div class="sc-num">{{ ov.project_total }}</div><div class="sc-label">书籍总数</div></div>
            <div class="stat-card"><div class="sc-num">{{ ov.ai_calls_7d }}</div><div class="sc-label">7 日 AI 调用</div></div>
            <div class="stat-card"><div class="sc-num">{{ fmtNum(ov.ai_tokens_7d) }}</div><div class="sc-label">7 日 AI tokens</div></div>
          </div>
          <div class="panel">
            <h3>30 日注册 / 活跃 / 字数</h3>
            <div class="chart-rows">
              <div v-for="d in dailyDates" :key="d" class="chart-row">
                <span class="cr-date">{{ d }}</span>
                <div class="cr-bars">
                  <span class="bar reg" :style="{ width: barW(regMap[d], maxReg) }" :title="`注册 ${regMap[d] || 0}`"></span>
                  <span class="bar act" :style="{ width: barW(actMap[d], maxAct) }" :title="`活跃 ${actMap[d] || 0}`"></span>
                </div>
                <span class="cr-words">{{ fmtNum(wordMap[d] || 0) }} 字</span>
              </div>
            </div>
            <div class="legend"><span class="dot reg"></span>注册 <span class="dot act"></span>活跃作者</div>
          </div>
          <div class="panel">
            <h3>30 日 AI 用量（按模型）</h3>
            <table class="admin-table" v-if="aiByModel.length">
              <tr><th>模型</th><th>调用</th><th>tokens</th><th>费用估算</th></tr>
              <tr v-for="m in aiByModel" :key="m.model_id">
                <td>{{ m.model_id }}</td><td>{{ m.calls }}</td><td>{{ fmtNum(m.tokens) }}</td><td>¥{{ m.cost.toFixed(2) }}</td>
              </tr>
            </table>
            <p v-else class="empty">暂无 AI 调用记录</p>
            <table class="admin-table" v-if="aiByMode.length">
              <tr><th>功能</th><th>调用</th></tr>
              <tr v-for="m in aiByMode" :key="m.ai_mode"><td>{{ m.ai_mode || '未分类' }}</td><td>{{ m.calls }}</td></tr>
            </table>
          </div>
        </template>
      </template>

      <!-- 用户管理 -->
      <template v-else-if="tab === 'users'">
        <div class="toolbar">
          <input v-model="userQ" class="input" placeholder="搜索邮箱 / 昵称 / user_id" @keyup.enter="loadUsers(1)" />
          <button class="btn primary" @click="loadUsers(1)">搜索</button>
        </div>
        <table class="admin-table">
          <tr><th>ID</th><th>邮箱</th><th>昵称</th><th>套餐</th><th>状态</th><th>书籍</th><th>注册时间</th><th>操作</th></tr>
          <tr v-for="u in users" :key="u.user_id">
            <td>{{ u.id }}</td>
            <td>{{ u.email }}<span v-if="u.is_admin" class="tag admin">管理员</span></td>
            <td>{{ u.nickname || '—' }}</td>
            <td>
              <select :value="u.plan" @change="setPlan(u, ($event.target as HTMLSelectElement).value)">
                <option value="free">free</option><option value="vip">vip</option><option value="svip">svip</option>
              </select>
            </td>
            <td><span class="tag" :class="u.status === 1 ? 'ok' : 'bad'">{{ u.status === 1 ? '正常' : u.status === 0 ? '禁用' : '注销' }}</span></td>
            <td>{{ u.project_count }}</td>
            <td>{{ fmtDate(u.created_at) }}</td>
            <td class="ops">
              <button v-if="u.status === 1" class="btn sm" @click="setStatus(u, 0)">禁用</button>
              <button v-else-if="u.status === 0" class="btn sm" @click="setStatus(u, 1)">恢复</button>
              <button class="btn sm" @click="resetPwd(u)">重置密码</button>
            </td>
          </tr>
        </table>
        <div class="pager" v-if="userTotal > 20">
          <button class="btn sm" :disabled="userPage <= 1" @click="loadUsers(userPage - 1)">上一页</button>
          <span>{{ userPage }} / {{ Math.ceil(userTotal / 20) }}（共 {{ userTotal }} 人）</span>
          <button class="btn sm" :disabled="userPage >= Math.ceil(userTotal / 20)" @click="loadUsers(userPage + 1)">下一页</button>
        </div>
      </template>

      <!-- 内容监管 -->
      <template v-else-if="tab === 'content'">
        <div class="toolbar">
          <input v-model="projQ" class="input" placeholder="搜索书名 / 作者邮箱" @keyup.enter="loadProjects(1)" />
          <button class="btn primary" @click="loadProjects(1)">搜索</button>
        </div>
        <table class="admin-table">
          <tr><th>ID</th><th>书名</th><th>作者</th><th>类型</th><th>状态</th><th>创建</th><th>操作</th></tr>
          <tr v-for="p in projects" :key="p.id">
            <td>{{ p.id }}</td>
            <td><a href="javascript:void(0)" @click="openDocs(p)">{{ p.name }}</a></td>
            <td>{{ p.owner_email }}</td>
            <td>{{ p.genre || '—' }}</td>
            <td><span class="tag" :class="p.status === 1 ? 'ok' : 'bad'">{{ p.status === 1 ? '正常' : '已下架' }}</span></td>
            <td>{{ fmtDate(p.created_at) }}</td>
            <td class="ops">
              <button class="btn sm" @click="openDocs(p)">章节</button>
              <button v-if="p.status === 1" class="btn sm danger" @click="setProjStatus(p, 0)">下架</button>
              <button v-else class="btn sm" @click="setProjStatus(p, 1)">恢复</button>
            </td>
          </tr>
        </table>

        <!-- 章节抽屉 -->
        <div v-if="docsProject" class="drawer-mask" @click.self="docsProject = null">
          <div class="drawer">
            <div class="drawer-head">
              <span>《{{ docsProject.name }}》章节</span>
              <button @click="docsProject = null">✕</button>
            </div>
            <div class="drawer-body">
              <div v-for="d in docs" :key="d.id" class="doc-row" @click="openDocContent(d)">
                <span class="doc-type">{{ d.type }}</span>
                <span class="doc-title">{{ d.title }}</span>
                <span class="doc-words">{{ d.words }} 字</span>
              </div>
            </div>
          </div>
        </div>
        <!-- 正文弹窗 -->
        <div v-if="viewDoc" class="drawer-mask" @click.self="viewDoc = null">
          <div class="drawer wide">
            <div class="drawer-head">
              <span>{{ viewDoc.title }}</span>
              <button @click="viewDoc = null">✕</button>
            </div>
            <div class="drawer-body"><pre class="doc-content">{{ viewDoc.content }}</pre></div>
          </div>
        </div>
      </template>

      <!-- 运营 -->
      <template v-else>
        <div class="panel">
          <h3>公告</h3>
          <div class="ann-form">
            <input v-model="annTitle" class="input" placeholder="公告标题" />
            <textarea v-model="annContent" class="input" rows="3" placeholder="公告内容（用户端可见）"></textarea>
            <button class="btn primary" @click="createAnn">发布</button>
          </div>
          <div v-for="a in anns" :key="a.id" class="ann-item">
            <div class="ann-head">
              <strong>{{ a.title }}</strong>
              <button class="btn sm danger" @click="delAnn(a.id)">删除</button>
            </div>
            <p>{{ a.content }}</p>
            <span class="ann-date">{{ fmtDate(a.created_at) }}</span>
          </div>
          <p v-if="!anns.length" class="empty">暂无公告</p>
        </div>
        <div class="panel">
          <h3>用户反馈 <span class="tag" v-if="openFeedbacks">{{ openFeedbacks }} 待处理</span></h3>
          <div v-for="f in feedbacks" :key="f.id" class="fb-item" :class="{ done: f.resolved }">
            <div class="fb-head">
              <span>{{ f.user_id ? `用户 ${f.user_id.slice(0, 8)}…` : '匿名' }}</span>
              <span>{{ fmtDate(f.created_at) }}</span>
            </div>
            <p>{{ f.content }}</p>
            <div class="fb-foot">
              <span v-if="f.contact" class="fb-contact">联系：{{ f.contact }}</span>
              <button v-if="!f.resolved" class="btn sm" @click="resolveFb(f.id)">标记已处理</button>
              <span v-else class="tag ok">已处理</span>
            </div>
          </div>
          <p v-if="!feedbacks.length" class="empty">暂无反馈</p>
        </div>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi, userApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const TABS = [
  { id: 'dashboard', icon: '📊', label: '看板' },
  { id: 'users', icon: '👥', label: '用户' },
  { id: 'content', icon: '📚', label: '内容' },
  { id: 'ops', icon: '📢', label: '运营' },
] as const
type TabId = typeof TABS[number]['id']
const tab = ref<TabId>('dashboard')

const isAdmin = ref(false)
const userEmail = ref('')

// —— 看板 ——
const ov = ref<Record<string, number>>({})
const regs = ref<{ date: string; n: number }[]>([])
const acts = ref<{ date: string; n: number }[]>([])
const words = ref<{ date: string; n: number }[]>([])
const aiByModel = ref<{ model_id: string; calls: number; tokens: number; cost: number }[]>([])
const aiByMode = ref<{ ai_mode: string; calls: number }[]>([])

const regMap = computed(() => Object.fromEntries(regs.value.map(r => [r.date, r.n])))
const actMap = computed(() => Object.fromEntries(acts.value.map(r => [r.date, r.n])))
const wordMap = computed(() => Object.fromEntries(words.value.map(r => [r.date, r.n])))
const dailyDates = computed(() => regs.value.map(r => r.date))
const maxReg = computed(() => Math.max(1, ...regs.value.map(r => r.n)))
const maxAct = computed(() => Math.max(1, ...acts.value.map(r => r.n)))
function barW(v: number | undefined, max: number) { return Math.max(2, Math.round(((v || 0) / max) * 100)) + '%' }

// —— 用户 ——
const userQ = ref('')
const users = ref<any[]>([])
const userPage = ref(1)
const userTotal = ref(0)

// —— 内容 ——
const projQ = ref('')
const projects = ref<any[]>([])
const docsProject = ref<any>(null)
const docs = ref<any[]>([])
const viewDoc = ref<any>(null)

// —— 运营 ——
const annTitle = ref('')
const annContent = ref('')
const anns = ref<any[]>([])
const feedbacks = ref<any[]>([])
const openFeedbacks = computed(() => feedbacks.value.filter(f => !f.resolved).length)

function fmtNum(n: number) {
  if (n >= 10000000) return (n / 10000000).toFixed(1) + '千万'
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return String(n ?? 0)
}
function fmtDate(s: string) { return s ? s.slice(0, 10) : '—' }

onMounted(async () => {
  try {
    const me = await userApi.me()
    userEmail.value = me.data?.data?.email || ''
  } catch {}
  loadDashboard()
})

async function checkPerm(): Promise<boolean> {
  // 无权限时第一个请求会 403，统一在这里给出提示
  if (isAdmin.value) return true
  try {
    await adminApi.statsOverview()
    isAdmin.value = true
    return true
  } catch (e: any) {
    if (e?.response?.status === 403) {
      isAdmin.value = false
      return false
    }
    throw e
  }
}

async function loadDashboard() {
  if (!(await checkPerm())) return
  const [o, d, a] = await Promise.all([
    adminApi.statsOverview(), adminApi.statsDaily(), adminApi.statsAIUsage(),
  ])
  ov.value = o.data?.data || {}
  regs.value = d.data?.data?.registrations || []
  acts.value = d.data?.data?.actives || []
  words.value = d.data?.data?.words || []
  aiByModel.value = a.data?.data?.by_model || []
  aiByMode.value = a.data?.data?.by_mode || []
}

async function loadUsers(page = 1) {
  if (!(await checkPerm())) return
  const res = await adminApi.listUsers({ q: userQ.value, page })
  users.value = res.data?.data?.list || []
  userTotal.value = res.data?.data?.total || 0
  userPage.value = page
}

async function setStatus(u: any, status: number) {
  await adminApi.setUserStatus(u.user_id, status)
  toast.success(status === 1 ? '已恢复' : '已禁用')
  loadUsers(userPage.value)
}

async function setPlan(u: any, plan: string) {
  await adminApi.setUserPlan(u.user_id, plan)
  toast.success(`已改为 ${plan}`)
  u.plan = plan
}

async function resetPwd(u: any) {
  const pwd = prompt(`给 ${u.email} 设置新密码（至少 6 位）：`)
  if (!pwd) return
  if (pwd.length < 6) { toast.error('密码至少 6 位'); return }
  await adminApi.resetUserPassword(u.user_id, pwd)
  toast.success('密码已重置')
}

async function loadProjects(page = 1) {
  if (!(await checkPerm())) return
  const res = await adminApi.listProjects({ q: projQ.value, page })
  projects.value = res.data?.data?.list || []
}

async function openDocs(p: any) {
  const res = await adminApi.listProjectDocuments(p.id)
  docs.value = res.data?.data || []
  docsProject.value = p
}

async function openDocContent(d: any) {
  const res = await adminApi.getDocumentContent(d.id)
  viewDoc.value = res.data?.data
}

async function setProjStatus(p: any, status: number) {
  if (status === 0 && !confirm(`确认下架《${p.name}》？作者将无法访问。`)) return
  await adminApi.setProjectStatus(p.id, status)
  toast.success(status === 1 ? '已恢复' : '已下架')
  loadProjects()
}

async function loadOps() {
  const [a, f] = await Promise.all([adminApi.listAnnouncements(), adminApi.listFeedbacks()])
  anns.value = a.data?.data || []
  feedbacks.value = f.data?.data?.list || []
}

async function createAnn() {
  if (!annTitle.value.trim()) { toast.error('标题不能为空'); return }
  await adminApi.createAnnouncement({ title: annTitle.value, content: annContent.value })
  annTitle.value = ''; annContent.value = ''
  toast.success('公告已发布')
  loadOps()
}

async function delAnn(id: number) {
  if (!confirm('删除这条公告？')) return
  await adminApi.deleteAnnouncement(id)
  loadOps()
}

async function resolveFb(id: number) {
  await adminApi.resolveFeedback(id)
  loadOps()
}

// tab 切换时按需加载
import { watch } from 'vue'
watch(tab, t => {
  if (t === 'users' && !users.value.length) loadUsers(1)
  if (t === 'content' && !projects.value.length) loadProjects(1)
  if (t === 'ops' && !anns.value.length) loadOps()
})
</script>

<style scoped>
.admin-page { min-height: 100vh; background: var(--bg-primary, #f6f6f8); }
.admin-header {
  display: flex; align-items: center; gap: 20px;
  padding: 14px 28px; background: var(--bg-secondary, #fff);
  border-bottom: 1px solid var(--border, #e5e5ea);
  position: sticky; top: 0; z-index: 10;
}
.admin-header h1 { font-size: 17px; margin: 0; color: var(--text-primary); }
.admin-tabs { display: flex; gap: 4px; flex: 1; }
.admin-tab {
  border: none; background: none; padding: 8px 16px; border-radius: 8px;
  cursor: pointer; font-size: 14px; color: var(--text-secondary);
}
.admin-tab:hover { background: var(--bg-tertiary, #f0f0f3); }
.admin-tab.active { background: var(--accent, #007aff); color: #fff; }
.btn-exit { border: none; background: none; color: var(--text-secondary); cursor: pointer; font-size: 13px; }
.btn-exit:hover { color: var(--text-primary); }
.admin-main { max-width: 1100px; margin: 0 auto; padding: 24px 28px 60px; }
.no-perm {
  padding: 60px 20px; text-align: center; color: var(--text-secondary);
  background: var(--bg-secondary, #fff); border-radius: 14px;
}
.stat-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin-bottom: 20px; }
.stat-card {
  background: var(--bg-secondary, #fff); border-radius: 12px; padding: 16px;
  text-align: center; border: 1px solid var(--border, #eee);
}
.sc-num { font-size: 26px; font-weight: 700; color: var(--accent, #007aff); font-variant-numeric: tabular-nums; }
.sc-label { font-size: 12px; color: var(--text-tertiary); margin-top: 4px; }
.panel {
  background: var(--bg-secondary, #fff); border-radius: 14px; padding: 18px 20px;
  border: 1px solid var(--border, #eee); margin-bottom: 18px;
}
.panel h3 { font-size: 15px; margin: 0 0 14px; color: var(--text-primary); }
.chart-rows { display: flex; flex-direction: column; gap: 3px; max-height: 320px; overflow-y: auto; }
.chart-row { display: flex; align-items: center; gap: 10px; font-size: 12px; }
.cr-date { width: 44px; color: var(--text-tertiary); flex-shrink: 0; }
.cr-bars { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.bar { height: 6px; border-radius: 3px; min-width: 2px; }
.bar.reg { background: #34c759; }
.bar.act { background: #007aff; }
.cr-words { width: 70px; text-align: right; color: var(--text-tertiary); flex-shrink: 0; }
.legend { margin-top: 10px; font-size: 12px; color: var(--text-tertiary); display: flex; gap: 14px; align-items: center; }
.dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; margin-right: 4px; vertical-align: -1px; }
.dot.reg { background: #34c759; } .dot.act { background: #007aff; }
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; }
.toolbar .input { flex: 1; }
.input {
  padding: 8px 12px; border: 1px solid var(--border, #e5e5ea); border-radius: 8px;
  font-size: 14px; outline: none; background: var(--bg-secondary, #fff); color: var(--text-primary);
  box-sizing: border-box;
}
.input:focus { border-color: var(--accent, #007aff); }
.admin-table { width: 100%; border-collapse: collapse; font-size: 13px; background: var(--bg-secondary, #fff); border-radius: 12px; overflow: hidden; }
.admin-table th { text-align: left; padding: 10px 12px; color: var(--text-tertiary); font-weight: 500; border-bottom: 1px solid var(--border, #eee); }
.admin-table td { padding: 10px 12px; border-bottom: 1px solid var(--border, #f0f0f3); color: var(--text-primary); }
.admin-table select { padding: 4px 6px; border-radius: 6px; border: 1px solid var(--border, #e5e5ea); background: var(--bg-secondary, #fff); }
.tag { font-size: 11px; padding: 2px 8px; border-radius: 10px; }
.tag.ok { background: rgba(52,199,89,.12); color: #248a3d; }
.tag.bad { background: rgba(255,59,48,.1); color: #d70015; }
.tag.admin { background: rgba(0,122,255,.1); color: #007aff; margin-left: 6px; }
.ops { display: flex; gap: 6px; }
.btn {
  border: 1px solid var(--border, #e0e0e0); background: transparent; padding: 7px 14px;
  border-radius: 8px; cursor: pointer; font-size: 13px; color: var(--text-primary);
}
.btn:hover { background: var(--bg-tertiary, #f5f5f7); }
.btn.primary { background: var(--accent, #007aff); color: #fff; border-color: var(--accent, #007aff); }
.btn.sm { padding: 4px 10px; font-size: 12px; }
.btn.danger { color: #d70015; border-color: rgba(255,59,48,.3); }
.btn.danger:hover { background: rgba(255,59,48,.06); }
.pager { display: flex; align-items: center; gap: 14px; justify-content: center; margin-top: 14px; font-size: 13px; color: var(--text-secondary); }
.empty { color: var(--text-tertiary); font-size: 13px; text-align: center; padding: 20px 0; }
.drawer-mask {
  position: fixed; inset: 0; background: rgba(0,0,0,.35); z-index: 100;
  display: flex; justify-content: flex-end;
}
.drawer {
  width: 420px; max-width: 90vw; background: var(--bg-secondary, #fff);
  height: 100%; display: flex; flex-direction: column; box-shadow: -8px 0 30px rgba(0,0,0,.15);
}
.drawer.wide { width: 560px; }
.drawer-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 18px; border-bottom: 1px solid var(--border, #eee); font-weight: 600;
}
.drawer-head button { border: none; background: none; cursor: pointer; font-size: 16px; }
.drawer-body { flex: 1; overflow-y: auto; padding: 12px 18px; }
.doc-row {
  display: flex; gap: 10px; align-items: center; padding: 8px 4px; cursor: pointer;
  border-bottom: 1px solid var(--border, #f0f0f3); font-size: 13px;
}
.doc-row:hover { background: var(--bg-tertiary, #f5f5f7); }
.doc-type { font-size: 11px; color: var(--text-tertiary); width: 52px; flex-shrink: 0; }
.doc-title { flex: 1; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.doc-words { color: var(--text-tertiary); font-size: 11px; }
.doc-content { white-space: pre-wrap; font-family: inherit; font-size: 14px; line-height: 1.8; margin: 0; }
.ann-form { display: flex; flex-direction: column; gap: 8px; margin-bottom: 18px; }
.ann-item { border-top: 1px solid var(--border, #f0f0f3); padding: 12px 0; }
.ann-head { display: flex; justify-content: space-between; align-items: center; }
.ann-head strong { font-size: 14px; color: var(--text-primary); }
.ann-item p { font-size: 13px; color: var(--text-secondary); margin: 6px 0; }
.ann-date { font-size: 11px; color: var(--text-tertiary); }
.fb-item { border-top: 1px solid var(--border, #f0f0f3); padding: 12px 0; }
.fb-item.done { opacity: .55; }
.fb-head { display: flex; justify-content: space-between; font-size: 12px; color: var(--text-tertiary); }
.fb-item p { font-size: 13px; color: var(--text-primary); margin: 8px 0; }
.fb-foot { display: flex; justify-content: space-between; align-items: center; }
.fb-contact { font-size: 12px; color: var(--text-secondary); }
</style>
