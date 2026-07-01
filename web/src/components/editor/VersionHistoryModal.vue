<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="modal version-modal" @click.stop>
      <div class="version-header">
        <div>
          <h3>版本历史</h3>
          <p>最近 3 个手动保存版本</p>
        </div>
        <button class="icon-btn" @click="close">×</button>
      </div>

      <div v-if="loading" class="empty-tip">加载中...</div>
      <div v-else class="version-layout">
        <div class="version-list">
          <div v-if="versions.length === 0" class="empty-tip">暂无手动保存版本</div>
          <div
            v-for="v in versions"
            :key="v.id"
            class="version-item"
            :class="{ active: selectedVersion?.id === v.id }"
            @click="$emit('select', v)"
          >
            <div class="version-time">{{ v.label || '未命名版本' }}</div>
            <div class="note">{{ formatDate(v.created_at) }} · {{ v.word_count || 0 }} 字</div>
          </div>
        </div>

        <div class="version-preview">
          <div v-if="!selectedVersion" class="empty-tip">选择一个版本查看预览</div>
          <template v-else>
            <div class="preview-toolbar">
              <span>{{ selectedVersion.label || '未命名版本' }} · {{ formatDate(selectedVersion.created_at) }}</span>
              <div class="preview-actions">
                <button class="btn" @click="$emit('compare')">对比当前</button>
                <button class="btn primary" @click="$emit('restore')" :disabled="restoring">
                  {{ restoring ? '恢复中...' : '恢复此版本' }}
                </button>
              </div>
            </div>
            <pre>{{ selectedVersion.content_snapshot || '' }}</pre>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: boolean
  loading: boolean
  restoring: boolean
  versions: any[]
  selectedVersion: any | null
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  select: [v: any]
  compare: []
  restore: []
}>()

function close() {
  emit('update:modelValue', false)
}

function formatDate(value: string) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString('zh-CN', { hour12: false })
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 500;
}

.modal.version-modal {
  width: 900px;
  max-width: 95vw;
  height: 600px;
  max-height: 85vh;
  background: var(--bg-secondary, #fff);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}

.version-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px 16px;
  border-bottom: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
}
.version-header h3 {
  margin: 0 0 4px 0;
  font-family: var(--font-serif);
  font-size: 18px;
  color: var(--text-primary, #1d1d1f);
}
.version-header p {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary, #86868b);
}
.icon-btn {
  background: transparent;
  border: none;
  font-size: 24px;
  color: var(--text-secondary, #86868b);
  cursor: pointer;
  padding: 0 8px;
  line-height: 1;
}
.icon-btn:hover { color: var(--text-primary, #1d1d1f); }

.version-layout {
  flex: 1;
  display: flex;
  min-height: 0;
}

.version-list {
  width: 240px;
  border-right: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  overflow-y: auto;
  padding: 8px 0;
}

.version-item {
  padding: 10px 20px;
  cursor: pointer;
  border-left: 3px solid transparent;
}
.version-item:hover { background: rgba(0, 0, 0, 0.02); }
.version-item.active {
  background: rgba(0, 122, 255, 0.06);
  border-left-color: var(--accent, #007aff);
}
.version-time {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary, #1d1d1f);
  margin-bottom: 2px;
}
.note {
  font-size: 11px;
  color: var(--text-tertiary, #aeaeb2);
}

.version-preview {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.empty-tip {
  color: var(--text-tertiary, #aeaeb2);
  font-size: 13px;
  padding: 24px;
  text-align: center;
}

.preview-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  border-bottom: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  font-size: 12px;
  color: var(--text-secondary, #86868b);
}
.preview-actions {
  display: flex;
  gap: 8px;
}

pre {
  flex: 1;
  overflow: auto;
  margin: 0;
  padding: 20px 24px;
  font-family: var(--font-serif);
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-primary, #1d1d1f);
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--bg-primary, #f5f5f7);
}

.btn {
  padding: 6px 14px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #fff);
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
}
.btn.primary {
  background: var(--accent, #007aff);
  color: white;
  border-color: var(--accent, #007aff);
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
