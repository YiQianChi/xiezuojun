<template>
  <!-- 全局随手记：非编辑器页右下速记笔（编辑器内有书内灵感页签，不重复） -->
  <div v-if="!open" class="qn-fab" title="随手记" @click="openCompose">
    <svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
  </div>
  <Teleport to="body">
    <Transition name="qn-fade">
      <div v-if="open" class="qn-overlay" @click.self="close">
        <!-- 记 -->
        <div v-if="view === 'compose'" class="qn-sheet">
          <div class="qn-top">
            <button class="qn-x" title="收起" @click="close">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
            </button>
            <span class="qn-title">随手记</span>
            <button class="qn-done" :disabled="saving" @click="save">{{ saving ? '…' : '记好了' }}</button>
          </div>
          <button class="qn-chip" @click="showPicker = !showPicker">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
            {{ pickedProject ? `《${pickedProject.name}》` : '未归档 · 可选挂到某书' }}
            <svg v-if="!showPicker" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M6 9l6 6 6-6"/></svg>
            <svg v-else width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M18 15l-6-6-6 6"/></svg>
          </button>
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
          <div class="qn-foot">
            <button class="qn-history" @click="openList">{{ quickCount }} 条速记 ›</button>
            <span class="qn-hint">{{ draft.length }}/500 · 草稿自动保存</span>
          </div>
        </div>
        <!-- 看列表 -->
        <div v-else class="qn-sheet qn-list-sheet">
          <div class="qn-top">
            <button class="qn-x" title="返回" @click="view = 'compose'">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M15 18l-6-6 6-6"/></svg>
            </button>
            <span class="qn-title">速记本 · {{ items.length }}</span>
            <button class="qn-x" title="收起" @click="close">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
            </button>
          </div>
          <div class="qn-cards">
            <p v-if="loadingList" class="qn-empty">加载中…</p>
            <p v-else-if="!items.length" class="qn-empty">还没有速记——右下角那支笔，随想随记</p>
            <div v-for="it in items" :key="it.id" class="qn-card">
              <p class="qn-card-t">{{ it.content }}</p>
              <div class="qn-card-m">
                <span>{{ fmtTime(it.created_at) }}</span>
                <span class="qn-acts">
                  <button @click="copy(it.content)">复制</button>
                  <button @click="startTransfer(it)">转入某书</button>
                  <button class="qn-del" @click="del(it)">删除</button>
                </span>
              </div>
            </div>
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
              <div class="qn-transfer-list">
                <p v-if="!projects.length" class="qn-picker-empty">还没有书</p>
                <button v-for="p in projects" :key="p.id" class="qn-transfer-row" @click="doTransfer(transferItem, p)">
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

const open = ref(false)
const view = ref<'compose' | 'list'>('compose')
const draft = ref('')
const saving = ref(false)
const taRef = ref<HTMLTextAreaElement | null>(null)

const projects = ref<{ id: number; name: string }[]>([])
const pickedProject = ref<{ id: number; name: string } | null>(null)
const showPicker = ref(false)

const items = ref<{ id: number; content: string; created_at: string }[]>([])
const loadingList = ref(false)
const transferItem = ref<{ id: number } | null>(null)
const quickCount = computed(() => items.value.length)

function fmtTime(iso: string) {
  const d = new Date(iso)
  const today = new Date()
  const same = d.toDateString() === today.toDateString()
  const hm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  return same ? `今天 ${hm}` : `${d.getMonth() + 1}/${d.getDate()} ${hm}`
}

async function openCompose() {
  open.value = true
  view.value = 'compose'
  try { draft.value = localStorage.getItem(DRAFT_KEY) || '' } catch {}
  await nextTick()
  taRef.value?.focus()
  if (!projects.value.length) {
    try {
      const res = await projectApi.list()
      projects.value = (res.data?.data || []).map((p: { id: number; name: string }) => ({ id: Number(p.id), name: p.name }))
    } catch { /* 挂书可选，拉不到书目不影响记 */ }
  }
  refreshCount()
}

async function refreshCount() {
  if (items.value.length || loadingList.value) return
  loadingList.value = true
  try {
    const res = await stickyApi.listQuick()
    items.value = res.data?.data || []
  } catch { /* 计数失败不拦主流程 */ }
  loadingList.value = false
}

function openList() {
  view.value = 'list'
  loadingList.value = true
  stickyApi.listQuick().then(res => { items.value = res.data?.data || [] }).catch(() => {}).finally(() => { loadingList.value = false })
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
    items.value = []
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
    transferItem.value = null
    toast.success(`已转入《${p.name}》灵感——编辑器右栏可见`)
  } catch {
    toast.info('转入失败，稍后再试')
  }
}

async function del(it: { id: number }) {
  try {
    await stickyApi.remove(it.id)
    items.value = items.value.filter(x => x.id !== it.id)
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
.qn-chip {
  align-self: flex-start; margin: 2px 16px 8px; cursor: pointer;
  font-size: 11px; color: #3f7a4e; background: rgba(129, 187, 142, .13);
  border: 1px solid rgba(129, 187, 142, .35); padding: 4px 11px; border-radius: 99px;
  display: inline-flex; align-items: center; gap: 5px;
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
  flex: 1; min-height: 180px; margin: 0 12px; background: #fff; border: none; outline: none;
  border-radius: 14px; padding: 14px; font-size: 16px; line-height: 1.8; color: #33302a;
  resize: none; font-family: inherit;
}
.qn-textarea::placeholder { color: #b8b8bd; }
.qn-foot { display: flex; justify-content: space-between; align-items: center; padding: 10px 16px 14px; }
.qn-history { background: none; border: none; cursor: pointer; font-size: 12px; color: #3f7a4e; padding: 2px; }
.qn-hint { font-size: 10px; color: #b0b0b5; }

.qn-list-sheet .qn-cards { overflow-y: auto; padding: 4px 12px 16px; }
.qn-empty { font-size: 12px; color: #aeaeb2; text-align: center; padding: 40px 0; }
.qn-card { background: #fff; border-radius: 12px; padding: 11px 13px; margin-bottom: 8px; }
.qn-card-t { font-size: 13px; line-height: 1.7; color: #33302a; white-space: pre-wrap; word-break: break-word; }
.qn-card-m { display: flex; align-items: baseline; gap: 8px; margin-top: 7px; font-size: 10px; color: #b0b0b5; flex-wrap: wrap; }
.qn-acts { margin-left: auto; display: flex; gap: 9px; flex-wrap: wrap; }
.qn-acts button { background: none; border: none; cursor: pointer; font-size: 10px; color: #86868b; padding: 0; }
.qn-acts button:hover { color: #3f7a4e; }
.qn-del:hover { color: #d06a5a !important; }

/* 选书底部面板（转入某书）：整行 ≥44px 点击区 */
.qn-transfer {
  position: absolute; left: 0; right: 0; bottom: 0; z-index: 5;
  background: #fff; border-radius: 18px 18px 0 0;
  box-shadow: 0 -8px 30px rgba(0, 0, 0, .14);
  display: flex; flex-direction: column; max-height: 55%;
}
.qn-transfer-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 13px 16px 7px; font-size: 12px; font-weight: 700; color: #86868b;
}
.qn-transfer-x { background: none; border: none; color: #aeaeb2; cursor: pointer; padding: 4px; display: flex; }
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
