<template>
  <!-- 全局随手记：非编辑器页右下速记笔（编辑器内用右栏底部常驻速记框，不重复） -->
  <div v-if="!open" class="qn-fab" title="随手记" @click="openCompose">
    <svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
  </div>
  <Teleport to="body">
    <Transition name="qn-fade">
      <div v-if="open" class="qn-overlay" @click.self="close">
        <!-- 单页速记：上输入下时间流，无二级视图 -->
        <div class="qn-sheet">
          <div class="qn-top">
            <button class="qn-x" title="收起" @click="close">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
            </button>
            <span class="qn-title">随手记</span>
            <button class="qn-done" :disabled="saving || !draft.trim()" @click="save">{{ saving ? '…' : '记好了' }}</button>
          </div>
          <div class="qn-input-zone">
            <button class="qn-chip" @click="showPicker = !showPicker">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
              {{ pickedProject ? `《${pickedProject.name}》` : '未归档 · 可选挂到某书' }}
              <svg v-if="!showPicker" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M6 9l6 6 6-6"/></svg>
              <svg v-else width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M18 15l-6-6-6 6"/></svg>
            </button>
            <span class="qn-hint">{{ draft.length }}/500 · 自动存草稿</span>
          </div>
          <div v-if="showPicker" class="qn-picker">
            <button class="qn-opt" :class="{ on: !pickedProject }" @click="pickedProject = null; showPicker = false">未归档（之后再转）</button>
            <button v-for="p in projects" :key="p.id" class="qn-opt" :class="{ on: pickedProject?.id === p.id }" @click="pickedProject = p; showPicker = false">《{{ p.name }}》</button>
            <p v-if="!projects.length" class="qn-picker-empty">还没有书，先记到未归档</p>
          </div>
          <textarea
            ref="taRef"
            v-model="draft"
            class="qn-textarea"
            placeholder="想到什么写什么"
            maxlength="500"
            @keydown.esc="close"
          ></textarea>
          <div class="qn-stream-head">
            <span class="qn-stream-label">速记流 · {{ total }} 条</span>
            <div class="qn-stream-search">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#aeaeb2" stroke-width="1.6" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M20 20l-4-4"/></svg>
              <input v-model.trim="streamQ" placeholder="搜速记" @input="onSearch" />
            </div>
          </div>
          <div class="qn-cards">
            <p v-if="loadingList" class="qn-empty">加载中…</p>
            <p v-else-if="!items.length" class="qn-empty">{{ streamQ ? '没找到这条速记' : '还没有速记——写完点「记好了」，都会攒在这里' }}</p>
            <template v-for="g in grouped" :key="g.label">
              <div class="qn-date">{{ g.label }}</div>
              <div v-for="it in g.items" :key="it.id" class="qn-card">
              <p class="qn-card-t">{{ it.content }}</p>
              <div class="qn-card-m">
                <span>{{ fmtTime(it.created_at) }}</span>
                <span class="qn-acts">
                  <button @click="copy(it.content)">复制</button>
                  <template v-if="lastBook">
                    <button class="qn-tolast" @click="doTransfer(it, lastBook)">→ {{ shortBook(lastBook.name) }}</button>
                    <button class="qn-swap" title="换一本书" @click="startTransfer(it)">▾</button>
                  </template>
                  <button v-else @click="startTransfer(it)">转入某书</button>
                  <button class="qn-del" @click="del(it)">删除</button>
                </span>
              </div>
              </div>
            </template>
            <button v-if="hasMore && !loadingList" class="qn-more" @click="loadMore">加载更多 · 还剩 {{ total - items.length }} 条</button>
          </div>
          <!-- 选书面板：整行大按钮，避免内联小字难点击 -->
          <Transition name="qn-sheet-up">
            <div v-if="transferItem" class="qn-transfer">
              <div class="qn-transfer-head">
                <span>转入哪本书的灵感</span>
                <button class="qn-transfer-x" @click="transferItem = null">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
                </button>
              </div>
              <div class="qn-transfer-search">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#aeaeb2" stroke-width="1.6" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M20 20l-4-4"/></svg>
                <input v-model.trim="searchQ" placeholder="搜书名" />
              </div>
              <div class="qn-transfer-list">
                <p v-if="!filteredProjects.length" class="qn-picker-empty">{{ searchQ ? '没找到这本书' : '还没有书' }}</p>
                <button v-for="p in filteredProjects" :key="p.id" class="qn-transfer-row" @click="doTransfer(transferItem!, p)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#81bb8e" stroke-width="1.6"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
                  {{ p.name }}
                </button>
              </div>
            </div>
          </Transition>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { stickyApi, projectApi } from '../core/api'
import { toast } from '../composables/useToast'

const DRAFT_KEY = 'xzj-quick-draft'
const LAST_BOOK_KEY = 'xzj-quick-last-book'

const open = ref(false)
const draft = ref('')
const saving = ref(false)
const taRef = ref<HTMLTextAreaElement | null>(null)

const projects = ref<{ id: number; name: string }[]>([])
const pickedProject = ref<{ id: number; name: string } | null>(null)
const showPicker = ref(false)

const items = ref<{ id: number; content: string; created_at: string }[]>([])
const loadingList = ref(false)
const transferItem = ref<{ id: number } | null>(null)
const searchQ = ref('')
const streamQ = ref('')
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null
const hasMore = computed(() => items.value.length < total.value)
// 日期分组（今天/昨天/M月D日）；跨页拼接后同日合并
const grouped = computed(() => {
  const groups: { label: string; items: { id: number; content: string; created_at: string }[] }[] = []
  for (const it of items.value) {
    const label = dateLabel(it.created_at)
    const lastG = groups[groups.length - 1]
    if (lastG && lastG.label === label) lastG.items.push(it)
    else groups.push({ label, items: [it] })
  }
  return groups
})
function dateLabel(iso: string) {
  const d = new Date(iso)
  const now = new Date()
  const day = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const diff = (day(now) - day(d)) / 86400000
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  return `${d.getMonth() + 1}月${d.getDate()}日`
}
// 上次转入的书：一键直达，换书才弹面板（▾）
const lastBook = ref<{ id: number; name: string } | null>(null)
try {
  const raw = localStorage.getItem(LAST_BOOK_KEY)
  if (raw) lastBook.value = JSON.parse(raw)
} catch {}
function rememberBook(p: { id: number; name: string }) {
  lastBook.value = p
  try { localStorage.setItem(LAST_BOOK_KEY, JSON.stringify(p)) } catch {}
}
function shortBook(name: string) {
  return name.length > 6 ? name.slice(0, 6) + '…' : name
}
const filteredProjects = computed(() =>
  searchQ.value ? projects.value.filter(p => p.name.includes(searchQ.value)) : projects.value
)

function fmtTime(iso: string) {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

async function openCompose() {
  open.value = true
  try { draft.value = localStorage.getItem(DRAFT_KEY) || '' } catch {}
  await nextTick()
  taRef.value?.focus()
  if (!projects.value.length) {
    try {
      const res = await projectApi.list()
      projects.value = (res.data?.data || []).map((p: { id: number; name: string }) => ({ id: Number(p.id), name: p.name }))
    } catch { /* 挂书可选，拉不到书目不影响记 */ }
  }
  loadItems()
}

let curPage = 1
// 速记流与输入区同屏；量大走分页 + 搜索（服务端 LIKE）
function loadItems(reset = true, page = 1) {
  loadingList.value = true
  stickyApi.listQuick({ q: streamQ.value || undefined, page, page_size: 50 }).then(res => {
    const list = res.data?.data || []
    items.value = reset ? list : items.value.concat(list)
    total.value = res.data?.total ?? list.length
    curPage = page
  }).catch(() => {}).finally(() => { loadingList.value = false })
}
function loadMore() {
  loadItems(false, curPage + 1)
}
function onSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadItems(true), 300)
}

function close() {
  open.value = false
  showPicker.value = false
  transferItem.value = null
}

watch(draft, v => {
  try { v ? localStorage.setItem(DRAFT_KEY, v) : localStorage.removeItem(DRAFT_KEY) } catch {}
})

async function save() {
  const text = draft.value.trim()
  if (!text || saving.value) return
  saving.value = true
  try {
    await stickyApi.create({
      project_id: pickedProject.value?.id || 0,
      type: pickedProject.value ? 'insp' : 'quick',
      content: text,
    })
    draft.value = ''
    loadItems()
    toast.success(pickedProject.value ? `已记进《${pickedProject.value.name}》灵感` : '已记下')
    close()
  } catch {
    toast.info('保存失败，草稿还在，稍后再试')
  }
  saving.value = false
}

function copy(text: string) {
  navigator.clipboard?.writeText(text).then(() => toast.success('已复制')).catch(() => {})
}

function startTransfer(it: { id: number }) {
  transferItem.value = it
  searchQ.value = ''
  if (!projects.value.length) {
    projectApi.list().then(res => {
      projects.value = (res.data?.data || []).map((p: { id: number; name: string }) => ({ id: Number(p.id), name: p.name }))
    }).catch(() => {})
  }
}

async function doTransfer(it: { id: number }, p: { id: number; name: string }) {
  try {
    await stickyApi.update(it.id, { project_id: p.id })
    items.value = items.value.filter(x => x.id !== it.id)
    total.value = Math.max(0, total.value - 1)
    transferItem.value = null
    rememberBook(p)
    toast.success(`已转入《${p.name}》灵感——编辑器右栏可见`)
  } catch {
    // 上次的书可能已被删（服务端校验拒绝）：清默认并让用户重选
    if (lastBook.value?.id === p.id) {
      lastBook.value = null
      try { localStorage.removeItem(LAST_BOOK_KEY) } catch {}
    }
    toast.info('转入失败，重新选一本书')
    startTransfer(it)
  }
}

async function del(it: { id: number }) {
  try {
    await stickyApi.remove(it.id)
    items.value = items.value.filter(x => x.id !== it.id)
    total.value = Math.max(0, total.value - 1)
  } catch { /* 下次打开自然对齐 */ }
}
</script>

<style scoped>
.qn-fab {
  position: fixed; right: 18px; bottom: calc(20px + env(safe-area-inset-bottom));
  width: 48px; height: 48px; border-radius: 50%; z-index: 700; cursor: pointer;
  background: linear-gradient(135deg, #8fc59b, #6aa877);
  box-shadow: 0 6px 16px rgba(106, 168, 119, .45);
  display: flex; align-items: center; justify-content: center;
  transition: transform .15s ease;
}
.qn-fab:hover { transform: scale(1.06); }
.qn-fab::after {
  content: ""; position: absolute; inset: -7px; border-radius: 50%;
  border: 1.5px solid rgba(129, 187, 142, .5);
  animation: qn-pulse 2.4s ease-out infinite; pointer-events: none;
}
@keyframes qn-pulse { 0% { transform: scale(.85); opacity: 1 } 100% { transform: scale(1.25); opacity: 0 } }

.qn-overlay {
  position: fixed; inset: 0; z-index: 1000; background: rgba(20, 22, 24, .42);
  display: flex; align-items: flex-end; justify-content: center;
}
.qn-sheet {
  width: 100%; max-width: 560px; background: #f7f7f5; border-radius: 20px 20px 0 0;
  display: flex; flex-direction: column; max-height: 92vh; padding-bottom: env(safe-area-inset-bottom);
  position: relative; overflow: hidden;
}
@media (min-width: 700px) {
  .qn-overlay { align-items: center; padding: 20px; }
  .qn-sheet { border-radius: 20px; }
}
.qn-top { display: flex; align-items: center; justify-content: space-between; padding: 14px 16px 8px; }
.qn-title { font-size: 15px; font-weight: 700; color: #33302a; }
.qn-x { background: none; border: none; color: #aeaeb2; cursor: pointer; padding: 4px; display: flex; }
.qn-done {
  border: none; cursor: pointer; font-size: 13px; font-weight: 700; color: #fff;
  background: linear-gradient(135deg, #8fc59b, #6aa877); border-radius: 99px; padding: 7px 18px;
  box-shadow: 0 4px 10px rgba(106, 168, 119, .35);
}
.qn-done:disabled { opacity: .6; }
.qn-input-zone { display: flex; align-items: center; justify-content: space-between; padding: 2px 16px 8px; gap: 8px; }
.qn-chip {
  cursor: pointer; margin: 0;
  font-size: 11px; color: #3f7a4e; background: rgba(129, 187, 142, .13);
  border: 1px solid rgba(129, 187, 142, .35); padding: 4px 11px; border-radius: 99px;
  display: inline-flex; align-items: center; gap: 5px; max-width: 70%; overflow: hidden; white-space: nowrap;
}
.qn-picker { margin: 0 16px 8px; background: #fff; border-radius: 12px; padding: 6px; max-height: 180px; overflow-y: auto; }
.qn-opt {
  display: block; width: 100%; text-align: left; background: none; border: none; cursor: pointer;
  font-size: 12px; color: #33302a; padding: 8px 10px; border-radius: 8px;
}
.qn-opt:hover { background: #f2f5f2; }
.qn-opt.on { color: #3f7a4e; font-weight: 700; }
.qn-picker-empty { font-size: 11px; color: #aeaeb2; padding: 8px 10px; }
.qn-textarea {
  min-height: 108px; margin: 0 12px; background: #fff; border: none; outline: none;
  border-radius: 14px; padding: 13px; font-size: 16px; line-height: 1.75; color: #33302a;
  resize: none; font-family: inherit; flex: 0 0 auto;
}
.qn-textarea::placeholder { color: #b8b8bd; }
.qn-hint { font-size: 10px; color: #b0b0b5; white-space: nowrap; }
.qn-stream-head { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px 7px; gap: 8px; }
.qn-stream-label { font-size: 11px; font-weight: 700; color: #86868b; white-space: nowrap; }
.qn-stream-search {
  display: flex; align-items: center; gap: 6px; background: #fff; border-radius: 99px;
  padding: 6px 12px; flex: 1; max-width: 180px;
}
.qn-stream-search input { border: none; outline: none; background: none; font-size: 12px; flex: 1; font-family: inherit; min-width: 0; }
.qn-date { font-size: 10px; color: #b0b0b5; padding: 8px 6px 5px; font-weight: 600; }
.qn-more {
  display: block; margin: 4px auto 8px; background: none; cursor: pointer;
  border: 1px solid rgba(129, 187, 142, .4); color: #3f7a4e; font-size: 11px;
  border-radius: 99px; padding: 7px 16px;
}
.qn-cards { flex: 1; overflow-y: auto; padding: 0 12px 16px; }
.qn-empty { font-size: 12px; color: #aeaeb2; text-align: center; padding: 40px 0; }
.qn-card { background: #fff; border-radius: 12px; padding: 11px 13px; margin-bottom: 8px; }
.qn-card-t { font-size: 13px; line-height: 1.7; color: #33302a; white-space: pre-wrap; word-break: break-word; }
.qn-card-m { display: flex; align-items: baseline; gap: 8px; margin-top: 7px; font-size: 10px; color: #b0b0b5; flex-wrap: wrap; }
.qn-acts { margin-left: auto; display: flex; gap: 9px; flex-wrap: wrap; }
.qn-acts button { background: none; border: none; cursor: pointer; font-size: 10px; color: #86868b; padding: 6px 1px; }
.qn-acts button:hover { color: #3f7a4e; }
.qn-tolast { color: #3f7a4e !important; font-weight: 700; font-size: 11px !important; }
.qn-swap { font-size: 10px !important; padding: 0 2px !important; }
.qn-del:hover { color: #d06a5a !important; }

/* 选书底部面板（换书 ▾ 时弹）：搜索 + 整行 ≥44px 点击区 */
.qn-transfer {
  position: absolute; left: 0; right: 0; bottom: 0; z-index: 5;
  background: #fff; border-radius: 18px 18px 0 0;
  box-shadow: 0 -8px 30px rgba(0, 0, 0, .14);
  display: flex; flex-direction: column; max-height: 62%;
}
.qn-transfer-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 13px 16px 7px; font-size: 12px; font-weight: 700; color: #86868b;
}
.qn-transfer-x { background: none; border: none; color: #aeaeb2; cursor: pointer; padding: 4px; display: flex; }
.qn-transfer-search {
  display: flex; align-items: center; gap: 7px; margin: 4px 14px 8px;
  background: #f5f5f7; border-radius: 10px; padding: 8px 12px;
}
.qn-transfer-search input { border: none; outline: none; background: none; font-size: 13px; flex: 1; font-family: inherit; }
.qn-transfer-list { overflow-y: auto; padding: 0 8px calc(10px + env(safe-area-inset-bottom)); }
.qn-transfer-row {
  display: flex; align-items: center; gap: 10px; width: 100%;
  background: none; border: none; cursor: pointer; text-align: left;
  font-size: 14px; color: #33302a; padding: 13px 10px; min-height: 44px;
  border-bottom: 1px solid #f2f2f4;
}
.qn-transfer-row:last-child { border-bottom: none; }
.qn-transfer-row:hover { background: #f5f8f5; }
.qn-sheet-up-enter-active, .qn-sheet-up-leave-active { transition: transform .2s ease; }
.qn-sheet-up-enter-from, .qn-sheet-up-leave-to { transform: translateY(100%); }

.qn-fade-enter-active, .qn-fade-leave-active { transition: opacity .16s ease; }
.qn-fade-enter-from, .qn-fade-leave-to { opacity: 0; }
</style>
