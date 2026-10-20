<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="modal" @click.stop>
      <div class="modal-head">
        <h3>{{ isFolder ? '文件夹属性' : '文件属性' }}</h3>
        <button class="close-btn" @click="close">✕</button>
      </div>

      <div class="modal-body">
        <!-- 图标（文件和文件夹都有） -->
        <div class="prop-row">
          <label class="prop-label">图标</label>
          <div class="icon-picker">
            <button
              v-for="ic in iconChoices"
              :key="ic"
              class="icon-chip"
              :class="{ active: selectedIcon === ic }"
              @click="selectedIcon = ic"
            >{{ ic }}</button>
          </div>
        </div>

        <!-- 文件专属：状态 + 目标字数 -->
        <template v-if="!isFolder">
          <div class="prop-row">
            <label class="prop-label">状态</label>
            <div class="status-picker">
              <button
                v-for="s in statusOptions"
                :key="s.value"
                class="status-chip"
                :class="{ active: selectedStatus === s.value }"
                @click="selectedStatus = s.value"
              >
                <span class="status-dot" :style="{ background: s.color }"></span>
                {{ s.label }}
              </button>
            </div>
          </div>
          <div class="prop-row half">
            <div class="prop-col">
              <label class="prop-label">目标字数</label>
              <input
                v-model.number="wordGoal"
                type="number"
                min="0"
                class="prop-input"
                placeholder="0 = 不限"
              />
            </div>
            <div class="prop-col" v-if="doc">
              <label class="prop-label">当前字数</label>
              <div class="prop-stat">
                <span class="stat-val">{{ doc.word_count || 0 }}</span>
                <span v-if="wordGoal > 0" class="stat-pct">{{ Math.min(100, Math.round(((doc.word_count || 0) / wordGoal) * 100)) }}%</span>
              </div>
            </div>
          </div>
        </template>
      </div>

      <div class="modal-foot">
        <button class="btn" @click="close">取消</button>
        <button class="btn primary" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  doc: any
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  save: [data: { icon?: string; doc_status?: string; word_goal?: number }]
}>()

const isFolder = computed(() => {
  if (!props.doc) return false
  const t = props.doc.type
  return t === 'volume' || t === 'outline' || t === 'characters' || t === 'worldbuilding' || t === 'storyline' || t === 'chapter_outline'
})

const selectedIcon = ref('')
const selectedStatus = ref('draft')
const wordGoal = ref(0)

const statusOptions = [
  { value: 'draft', label: '草稿', color: '#8e8e93' },
  { value: 'revising', label: '修改中', color: '#f5a623' },
  { value: 'final', label: '定稿', color: '#34c759' },
]

const iconChoices = [
  '📁', '📋', '👥', '🌍', '📖', '📑', '📝', '📄',
  '🗂️', '🏷️', '⭐', '💡', '🔒', '⚔️', '🏰', '🗺️',
  '✒️', '✏️', '🖊️', '📚', '🎨', '🎬', '🎵', '💎',
  '🔥', '❄️', '🌙', '☀️', '⚡', '🌟', '🎭', '🧩',
]

watch(() => props.modelValue, (v) => {
  if (v && props.doc) {
    selectedIcon.value = props.doc.icon || ''
    selectedStatus.value = props.doc.doc_status || 'draft'
    wordGoal.value = props.doc.word_goal || 0
  }
})

function close() {
  emit('update:modelValue', false)
}

function save() {
  const data: any = {
    icon: selectedIcon.value,
  }
  if (!isFolder.value) {
    data.doc_status = selectedStatus.value
    data.word_goal = wordGoal.value || 0
  }
  emit('save', data)
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
.modal {
  width: 460px;
  max-width: 90vw;
  background: var(--bg-secondary, #fff);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
  display: flex;
  flex-direction: column;
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border, rgba(0,0,0,0.06));
}
.modal-head h3 {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 17px;
  font-weight: 600;
  color: var(--text-primary, #1d1d1f);
}
.close-btn {
  border: none;
  background: none;
  cursor: pointer;
  font-size: 16px;
  color: var(--text-tertiary, #86868b);
  padding: 2px 8px;
  border-radius: 6px;
}
.close-btn:hover { background: rgba(0,0,0,0.06); }

.modal-body {
  padding: 16px 20px;
  overflow-y: auto;
  max-height: 60vh;
}

.prop-row {
  margin-bottom: 16px;
}
.prop-row.half {
  display: flex;
  gap: 16px;
}
.prop-col {
  flex: 1;
  min-width: 0;
}
.prop-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary, #86868b);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
/* icon picker */
.icon-picker {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 5px;
}
.icon-chip {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #fff);
  font-size: 17px;
  cursor: pointer;
  transition: all 0.15s;
}
.icon-chip:hover { border-color: var(--accent, #007aff); }
.icon-chip.active {
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.08);
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.15);
}

/* status picker */
.status-picker {
  display: flex;
  gap: 8px;
}
.status-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #fff);
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
  transition: all 0.15s;
}
.status-chip.active {
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.08);
  color: var(--accent, #007aff);
}
.status-chip .status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* number input */
.prop-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  background: var(--bg-primary, #f5f5f7);
  color: var(--text-primary, #1d1d1f);
  transition: all 0.15s;
  box-sizing: border-box;
}
.prop-input:focus {
  border-color: var(--accent, #007aff);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

/* stat display */
.prop-stat {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-tertiary, #f5f5f7);
  border-radius: 8px;
}
.stat-val {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary, #1d1d1f);
  font-variant-numeric: tabular-nums;
}
.stat-pct {
  font-size: 12px;
  color: var(--text-secondary, #86868b);
}

.modal-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px 16px;
  border-top: 1px solid var(--border, rgba(0,0,0,0.06));
}
.btn {
  padding: 7px 18px;
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
</style>
