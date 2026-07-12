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
          <button class="destroy-btn" @click="openDestroy(item)">删除</button>
        </div>
      </div>
    </div>
    <div v-if="items.length >= MAX" class="trash-full-hint">回收站已满，新删除将自动淘汰最早的文件</div>

    <!-- 确认彻底删除弹窗 -->
    <div v-if="confirmDestroy" class="confirm-overlay" @click="confirmDestroy = null">
      <div class="confirm-modal" @click.stop>
        <div class="confirm-icon">⚠️</div>
        <h3 class="confirm-title">彻底删除</h3>
        <p class="confirm-desc">
          确定要彻底删除「{{ confirmDestroy.title }}」吗？
        </p>
        <p class="confirm-warn">⚠️ 删除后不可恢复</p>
        <p class="confirm-hint">请输入 <strong>确认删除</strong> 以继续</p>
        <input v-model="confirmText" class="confirm-input" placeholder="确认删除" />
        <div class="confirm-buttons">
          <button class="btn-cancel" @click="confirmDestroy = null; confirmText = ''">取消</button>
          <button class="btn-confirm-danger" :disabled="confirmText !== '确认删除'" @click="doDestroy">确认删除</button>
        </div>
      </div>
    </div>

    <!-- 确认清空回收站弹窗 -->
    <div v-if="confirmEmpty" class="confirm-overlay" @click="cancelEmpty">
      <div class="confirm-modal" @click.stop>
        <div class="confirm-icon">⚠️</div>
        <h3 class="confirm-title">清空回收站</h3>
        <p class="confirm-desc">确定要清空回收站吗？将删除全部 {{ items.length }} 个项目。</p>
        <p class="confirm-warn">⚠️ 删除后不可恢复</p>
        <p class="confirm-hint">请输入 <strong>确认删除</strong> 以继续</p>
        <input v-model="confirmText" class="confirm-input" placeholder="确认删除" />
        <div class="confirm-buttons">
          <button class="btn-cancel" @click="cancelEmpty">取消</button>
          <button class="btn-confirm-danger" :disabled="confirmText !== '确认删除'" @click="doEmpty">确认删除</button>
        </div>
      </div>
    </div>
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
const confirmDestroy = ref<any>(null)
const confirmText = ref('')
const confirmEmpty = ref(false)

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
  confirmText.value = ''
  confirmEmpty.value = true
}

function cancelEmpty() {
  confirmEmpty.value = false
  confirmText.value = ''
}

async function doEmpty() {
  try {
    await trashApi.empty(props.projectId)
    items.value = []
    toast.success('已清空回收站')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '清空失败')
  } finally {
    cancelEmpty()
  }
}

function openDestroy(item: any) {
  confirmText.value = ''
  confirmDestroy.value = item
}

async function doDestroy() {
  const item = confirmDestroy.value
  if (!item) return
  try {
    await trashApi.destroy({ project_id: Number(props.projectId), doc_id: item.id })
    toast.success(`已彻底删除「${item.title}」`)
    await load()
    emit('restored')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '删除失败')
  } finally {
    confirmDestroy.value = null
    confirmText.value = ''
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
.destroy-btn {
  border: 1px solid var(--border, rgba(0,0,0,0.1));
  background: transparent;
  color: var(--danger, #ff4d6d);
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 6px;
  cursor: pointer;
  margin-left: 4px;
  transition: all 0.1s;
}
.destroy-btn:hover {
  border-color: var(--danger, #ff4d6d);
  background: rgba(255,77,109,0.1);
}

/* 确认弹窗 */
.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.confirm-modal {
  background: var(--bg-primary, #fff);
  border-radius: 12px;
  padding: 28px 24px 20px;
  max-width: 360px;
  width: calc(100% - 48px);
  text-align: center;
  box-shadow: 0 8px 32px rgba(0,0,0,0.2);
}
.confirm-icon { font-size: 32px; margin-bottom: 8px; }
.confirm-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 8px; }
.confirm-desc { font-size: 13px; color: var(--text-secondary); margin: 0 0 6px; }
.confirm-warn { font-size: 13px; color: var(--danger, #ff4d6d); font-weight: 600; margin: 0 0 12px; }
.confirm-hint { font-size: 12px; color: var(--text-secondary); margin: 0 0 8px; }
.confirm-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border, rgba(0,0,0,0.1));
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
  margin-bottom: 16px;
}
.confirm-input:focus { border-color: var(--danger, #ff4d6d); }
.confirm-buttons { display: flex; gap: 8px; justify-content: center; }
.btn-cancel {
  padding: 8px 20px;
  border: 1px solid var(--border, rgba(0,0,0,0.1));
  background: transparent;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  color: var(--text-primary);
}
.btn-confirm-danger {
  padding: 8px 20px;
  border: none;
  background: var(--danger, #ff4d6d);
  color: #fff;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
}
.btn-confirm-danger:disabled {
  opacity: 0.4;
  cursor: not-allowed;
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
