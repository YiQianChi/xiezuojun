<template>
  <div class="trash-panel">
    <div class="trash-header">
      <span class="trash-title">🗑 回收站</span>
      <span class="trash-count">{{ items.length }}/{{ MAX }}</span>
      <button v-if="items.length" class="trash-empty-btn" @click="emptyAll">清空</button>
    </div>
    <div class="trash-list">
      <div v-if="loading" class="trash-empty">加载中...</div>
      <div v-else-if="items.length === 0" class="trash-empty">回收站为空</div>
      <div v-for="item in items" :key="item.id" class="trash-item">
        <span class="trash-icon">{{ typeIcon(item.type) }}</span>
        <div class="trash-info">
          <div class="trash-name">{{ item.title }}</div>
          <div class="trash-meta">
            {{ typeLabel(item.type) }} · {{ formatTime(item.updated_at) }}
            <span v-if="item.child_count" class="child-badge">含{{ item.child_count }}个子文件</span>
          </div>
        </div>
        <div class="trash-actions">
          <button class="restore-btn" @click="restore(item)">恢复</button>
        </div>
      </div>
    </div>
    <div v-if="items.length >= MAX" class="trash-full-hint">回收站已满，新删除将自动淘汰最早的文件</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { trashApi } from '../../core/api'
import { toast } from '../../composables/useToast'

const props = defineProps<{ projectId: string }>()
const emit = defineEmits<{ (e: 'restored'): void }>()

const MAX = 30
const items = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await trashApi.list(props.projectId)
    items.value = res.data?.data || []
  } catch { items.value = [] }
  loading.value = false
}

async function restore(item: any) {
  try {
    await trashApi.restore({ project_id: Number(props.projectId), doc_id: item.id })
    toast.success(`已恢复「${item.title}」`)
    await load()
    emit('restored')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '恢复失败')
  }
}

async function emptyAll() {
  if (!confirm('确定清空回收站？此操作不可撤销。')) return
  try {
    await trashApi.empty(props.projectId)
    items.value = []
    toast.success('已清空回收站')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '清空失败')
  }
}

function typeIcon(t: string) {
  return { volume: '📁', chapter: '📄', note: '📝', outline: '📋', characters: '👥', worldbuilding: '🌍', storyline: '📖', chapter_outline: '📑' }[t] || '📄'
}
function typeLabel(t: string) {
  return { volume: '文件夹', chapter: '章节', note: '笔记', outline: '大纲', characters: '人物', worldbuilding: '世界观', storyline: '故事主线', chapter_outline: '章节大纲' }[t] || '文档'
}
function formatTime(t: string) {
  if (!t) return ''
  const d = new Date(t)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

defineExpose({ load })
onMounted(load)
</script>

<style scoped>
.trash-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.trash-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  flex-shrink: 0;
}
.trash-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.trash-count {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-tertiary, rgba(0,0,0,0.04));
  padding: 2px 8px;
  border-radius: 10px;
}
.trash-empty-btn {
  margin-left: auto;
  border: none;
  background: transparent;
  color: var(--danger, #ff4d6d);
  font-size: 12px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
}
.trash-empty-btn:hover { background: rgba(255,77,109,0.1); }

.trash-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.trash-empty {
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
  padding: 40px 0;
}
.trash-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  transition: background 0.1s;
}
.trash-item:hover { background: var(--bg-hover, rgba(0,0,0,0.03)); }
.trash-icon { font-size: 16px; flex-shrink: 0; }
.trash-info { flex: 1; min-width: 0; }
.trash-name {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.trash-meta {
  font-size: 11px;
  color: var(--text-secondary);
}
.child-badge {
  margin-left: 4px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0,122,255,0.1);
  color: var(--accent, #007aff);
}
.trash-actions { flex-shrink: 0; }
.restore-btn {
  border: 1px solid var(--border, rgba(0,0,0,0.1));
  background: transparent;
  color: var(--text-primary);
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.1s;
}
.restore-btn:hover {
  border-color: var(--accent, #007aff);
  color: var(--accent, #007aff);
}
.trash-full-hint {
  padding: 6px 14px;
  font-size: 11px;
  color: var(--warn, #ff9500);
  text-align: center;
  border-top: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  flex-shrink: 0;
}
</style>
