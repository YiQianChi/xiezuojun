<template>
  <div class="thread-panel">
    <div class="thread-header">
      <span class="thread-title">伏笔追踪</span>
      <div class="thread-filter">
        <button
          v-for="f in filters"
          :key="f.key"
          class="filter-btn"
          :class="{ active: activeFilter === f.key }"
          @click="activeFilter = f.key"
        >{{ f.label }} <span class="filter-count">{{ counts[f.key] }}</span></button>
      </div>
    </div>

    <div class="thread-list">
      <div v-if="loading" class="thread-empty">加载中...</div>
      <div v-else-if="editingId === 0" class="thread-item editing">
        <div class="thread-edit-form">
          <select v-model="editForm.type" class="edit-select">
            <option value="foreshadowing">伏笔</option>
            <option value="hook">钩子</option>
            <option value="promise">承诺</option>
            <option value="suspense">悬念</option>
          </select>
          <input v-model="editForm.title" class="edit-input" placeholder="标题">
          <textarea v-model="editForm.description" class="edit-textarea" placeholder="描述（可选）" rows="2"></textarea>
          <div class="edit-row">
            <label>埋设章</label>
            <input v-model.number="editForm.planted_chapter" type="number" class="edit-num" placeholder="第几章">
            <label>回收章</label>
            <input v-model.number="editForm.resolved_chapter" type="number" class="edit-num" placeholder="第几章">
          </div>
          <div class="edit-row">
            <label>紧急度</label>
            <input v-model.number="editForm.urgency" type="range" min="0" max="100" class="edit-range">
            <span class="edit-urgency-val">{{ editForm.urgency || 0 }}</span>
          </div>
          <div class="edit-actions">
            <button class="btn-save" @click="saveEdit">保存</button>
            <button class="btn-cancel" @click="cancelEdit">取消</button>
          </div>
        </div>
      </div>
      <div v-else-if="filteredThreads.length === 0" class="thread-empty">
        暂无{{ activeFilter === 'open' ? '未回收' : activeFilter === 'resolved' ? '已回收' : '' }}伏笔
      </div>
      <div
        v-for="t in filteredThreads"
        :key="t.id"
        class="thread-item"
        :class="{ editing: editingId === t.id }"
      >
        <template v-if="editingId === t.id">
          <div class="thread-edit-form">
            <select v-model="editForm.type" class="edit-select">
              <option value="foreshadowing">伏笔</option>
              <option value="hook">钩子</option>
              <option value="promise">承诺</option>
              <option value="suspense">悬念</option>
            </select>
            <input v-model="editForm.title" class="edit-input" placeholder="标题">
            <textarea v-model="editForm.description" class="edit-textarea" placeholder="描述（可选）" rows="2"></textarea>
            <div class="edit-row">
              <label>埋设章</label>
              <input v-model.number="editForm.planted_chapter" type="number" class="edit-num" placeholder="第几章">
              <label>回收章</label>
              <input v-model.number="editForm.resolved_chapter" type="number" class="edit-num" placeholder="第几章">
            </div>
            <div class="edit-row">
              <label>紧急度</label>
              <input v-model.number="editForm.urgency" type="range" min="0" max="100" class="edit-range">
              <span class="edit-urgency-val">{{ editForm.urgency || 0 }}</span>
              <select v-model="editForm.status" class="edit-select">
                <option value="open">未回收</option>
                <option value="resolved">已回收</option>
                <option value="abandoned">已放弃</option>
              </select>
            </div>
            <div class="edit-actions">
              <button class="btn-save" @click="saveEdit">保存</button>
              <button class="btn-cancel" @click="cancelEdit">取消</button>
            </div>
          </div>
        </template>
        <template v-else>
          <div class="thread-item-header">
            <span class="thread-badge" :class="t.type">{{ typeLabel[t.type] || t.type }}</span>
            <span class="thread-item-title">{{ t.title }}</span>
            <span v-if="t.status === 'resolved'" class="thread-status resolved">✓</span>
            <span v-else-if="t.status === 'abandoned'" class="thread-status abandoned">—</span>
          </div>
          <div v-if="t.description" class="thread-desc">{{ t.description }}</div>
          <div class="thread-meta">
            <span v-if="t.planted_chapter" class="meta-chip">第{{ t.planted_chapter }}章埋</span>
            <span v-if="t.resolved_chapter" class="meta-chip resolved">第{{ t.resolved_chapter }}章收</span>
            <span class="meta-chip urgency" :class="urgencyClass(t.urgency || 0)">紧急 {{ t.urgency || 0 }}</span>
          </div>
          <div class="thread-actions">
            <button @click="startEdit(t)" class="action-btn">编辑</button>
            <button v-if="t.status === 'open'" @click="quickResolve(t)" class="action-btn">标记回收</button>
            <button @click="deleteThread(t)" class="action-btn danger">删除</button>
          </div>
        </template>
      </div>
    </div>

    <button class="thread-add" @click="startAdd">+ 添加伏笔</button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { threadApi, type StoryThread } from '../../core/api'

const props = defineProps<{ projectId: string | number }>()

const threads = ref<StoryThread[]>([])
const loading = ref(false)
const activeFilter = ref<'all' | 'open' | 'resolved' | 'abandoned'>('open')
const editingId = ref<number | null>(null)
const editForm = ref<Partial<StoryThread>>({})

const typeLabel: Record<string, string> = {
  foreshadowing: '伏笔',
  hook: '钩子',
  promise: '承诺',
  suspense: '悬念',
}

const filters = [
  { key: 'open' as const, label: '未回收' },
  { key: 'resolved' as const, label: '已回收' },
  { key: 'all' as const, label: '全部' },
]

const counts = computed(() => ({
  open: threads.value.filter((t: StoryThread) => t.status === 'open').length,
  resolved: threads.value.filter((t: StoryThread) => t.status === 'resolved').length,
  abandoned: threads.value.filter((t: StoryThread) => t.status === 'abandoned').length,
  all: threads.value.length,
}))

const filteredThreads = computed(() => {
  const list = activeFilter.value === 'all'
    ? threads.value
    : threads.value.filter((t: StoryThread) => t.status === activeFilter.value)
  return [...list].sort((a, b) => (b.urgency || 0) - (a.urgency || 0))
})

function urgencyClass(u: number) {
  if (u >= 80) return 'high'
  if (u >= 50) return 'mid'
  return 'low'
}

async function loadThreads() {
  loading.value = true
  try {
    const res = await threadApi.list(props.projectId)
    threads.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

function startEdit(t: StoryThread) {
  editingId.value = t.id!
  editForm.value = { ...t }
}

function startAdd() {
  editingId.value = 0
  editForm.value = {
    type: 'foreshadowing',
    title: '',
    description: '',
    status: 'open',
    urgency: 50,
    planted_chapter: null,
    resolved_chapter: null,
  }
}

function cancelEdit() {
  editingId.value = null
  editForm.value = {}
}

async function saveEdit() {
  const f = editForm.value
  if (!f.title?.trim()) return
  try {
    if (editingId.value === 0) {
      const res = await threadApi.create(props.projectId, f)
      threads.value.push(res.data.data)
    } else {
      await threadApi.update(editingId.value!, f)
      const idx = threads.value.findIndex((t: StoryThread) => t.id === editingId.value)
      if (idx >= 0) threads.value[idx] = { ...threads.value[idx], ...f } as StoryThread
    }
    cancelEdit()
  } catch (e) {
    console.error('save thread failed', e)
  }
}

async function quickResolve(t: StoryThread) {
  const chapter = prompt('回收于第几章？（留空则不填）')
  if (chapter === null) return
  const ch = chapter ? Number(chapter) : null
  await threadApi.update(t.id!, { status: 'resolved', resolved_chapter: ch })
  const idx = threads.value.findIndex((x: StoryThread) => x.id === t.id)
  if (idx >= 0) {
    threads.value[idx] = { ...threads.value[idx], status: 'resolved', resolved_chapter: ch }
  }
}

async function deleteThread(t: StoryThread) {
  if (!confirm('删除这条伏笔？')) return
  await threadApi.delete(t.id!)
  threads.value = threads.value.filter((x: StoryThread) => x.id !== t.id)
}

watch(() => props.projectId, loadThreads)
onMounted(loadThreads)
</script>

<style scoped>
.thread-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.thread-header {
  padding: 8px 12px 6px;
  border-bottom: 1px solid var(--border-subtle);
}
.thread-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}
.thread-filter {
  display: flex;
  gap: 4px;
  margin-top: 6px;
}
.filter-btn {
  border: none;
  background: transparent;
  padding: 3px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
}
.filter-btn.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-weight: 500;
}
.filter-count {
  opacity: 0.5;
}
.thread-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px 8px;
}
.thread-empty {
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
  padding: 16px;
}
.thread-item {
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 8px 10px;
  margin-bottom: 6px;
  background: var(--bg-primary);
}
.thread-item.editing {
  border-color: var(--accent, #007aff);
}
.thread-item-header {
  display: flex;
  align-items: center;
  gap: 6px;
}
.thread-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
  flex-shrink: 0;
}
.thread-badge.foreshadowing { background: rgba(255, 159, 10, 0.15); color: #b86e00; }
.thread-badge.hook { background: rgba(255, 69, 58, 0.15); color: #c0392b; }
.thread-badge.promise { background: rgba(0, 122, 255, 0.15); color: #005ec4; }
.thread-badge.suspense { background: rgba(88, 86, 214, 0.15); color: #4a48b8; }
.thread-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.thread-status.resolved { color: #34c759; font-size: 14px; }
.thread-status.abandoned { color: var(--text-tertiary); font-size: 14px; }
.thread-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
  line-height: 1.4;
}
.thread-meta {
  display: flex;
  gap: 4px;
  margin-top: 4px;
  flex-wrap: wrap;
}
.meta-chip {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(0,0,0,0.05);
  color: var(--text-secondary);
}
.meta-chip.resolved { background: rgba(52, 199, 89, 0.15); color: #1a8a3e; }
.meta-chip.urgency.high { background: rgba(255, 69, 58, 0.15); color: #c0392b; }
.meta-chip.urgency.mid { background: rgba(255, 159, 10, 0.15); color: #b86e00; }
.meta-chip.urgency.low { background: rgba(0,0,0,0.05); color: var(--text-tertiary); }
.thread-actions {
  display: flex;
  gap: 4px;
  margin-top: 6px;
}
.action-btn {
  border: none;
  background: transparent;
  padding: 2px 6px;
  font-size: 11px;
  color: var(--accent, #007aff);
  cursor: pointer;
  border-radius: 4px;
}
.action-btn:hover { background: rgba(0, 122, 255, 0.08); }
.action-btn.danger { color: #ff3b30; }
.action-btn.danger:hover { background: rgba(255, 59, 48, 0.08); }
.thread-add {
  margin: 6px 8px 8px;
  padding: 6px;
  border: 1px dashed var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--text-tertiary);
  font-size: 12px;
  cursor: pointer;
}
.thread-add:hover {
  color: var(--accent, #007aff);
  border-color: var(--accent, #007aff);
}
/* edit form */
.thread-edit-form {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.edit-select, .edit-input, .edit-textarea, .edit-num {
  padding: 4px 6px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 12px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
.edit-textarea { resize: none; }
.edit-row {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-secondary);
}
.edit-row label { flex-shrink: 0; }
.edit-num { width: 50px; flex: none; }
.edit-range { flex: 1; }
.edit-urgency-val { width: 24px; text-align: right; }
.edit-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
}
.btn-save, .btn-cancel {
  padding: 3px 10px;
  font-size: 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
.btn-save { background: var(--accent, #007aff); color: white; }
.btn-cancel { background: var(--bg-secondary); color: var(--text-secondary); }
</style>
