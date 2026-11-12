<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="modal" @click.stop>
      <div class="modal-head">
        <h3>{{ isFolder ? '文件夹属性' : '文件属性' }}</h3>
        <button class="close-btn" @click="close"><X :size="16"/></button>
      </div>

      <div class="modal-body">
        <!-- 基础信息：名称（可改） -->
        <div class="prop-row">
          <label class="prop-label">名称</label>
          <input
            v-model="title"
            class="prop-input"
            :placeholder="doc && doc.type === 'chapter' ? '留空 = 自动「第N章」编号' : '输入名称'"
          />
        </div>

        <!-- 基础信息：类型/创建/修改（只读，单列不换行） -->
        <div class="meta-list">
          <div class="meta-item"><span class="meta-k">类型</span><span class="meta-v">{{ typeLabel }}</span></div>
          <div class="meta-item" v-if="doc"><span class="meta-k">创建</span><span class="meta-v">{{ fmtDate(doc.created_at) }}</span></div>
          <div class="meta-item" v-if="doc"><span class="meta-k">最后修改</span><span class="meta-v">{{ fmtDate(doc.updated_at) }}</span></div>
        </div>

        <!-- 图标（文件和文件夹都有） -->
        <div class="prop-row">
          <label class="prop-label">图标</label>
          <div class="icon-picker">
            <button
              v-for="ic in NODE_ICONS"
              :key="ic.key"
              class="icon-chip"
              :class="{ active: selectedIcon === ic.key || EMOJI_TO_KEY[selectedIcon] === ic.key }"
              @click="selectedIcon = ic.key"
            ><component :is="ic.icon" :size="16"/></button>
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
import { X } from 'lucide-vue-next'
import { NODE_ICONS, EMOJI_TO_KEY } from '../nodeIcons'

const props = defineProps<{
  modelValue: boolean
  doc: any
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  save: [data: { icon?: string; title?: string; doc_status?: string; word_goal?: number }]
}>()

const isFolder = computed(() => {
  if (!props.doc) return false
  const t = props.doc.type
  return t === 'volume' || t === 'outline' || t === 'characters' || t === 'worldbuilding' || t === 'storyline' || t === 'chapter_outline'
})

const selectedIcon = ref('')
const selectedStatus = ref('draft')
const wordGoal = ref(0)
const title = ref('')

const TYPE_LABELS: Record<string, string> = {
  volume: '卷', folder: '文件夹', chapter: '章节', note: '笔记', outline: '大纲',
  characters: '人物', character: '角色卡', worldbuilding: '世界观', storyline: '故事线',
  chapter_outline: '章节大纲',
}
const typeLabel = computed(() => TYPE_LABELS[props.doc?.type] || props.doc?.type || '—')

function fmtDate(s?: string) {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(+d)) return '—'
  const pad = (n: number) => String(n).padStart(2, '0')
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) {
    const min = Math.floor((now.getTime() - d.getTime()) / 60000)
    if (min < 1) return '刚刚'
    if (min < 60) return `${min} 分钟前`
    return `今天 ${hm}`
  }
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日 ${hm}`
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

const statusOptions = [
  { value: 'draft', label: '草稿', color: '#8e8e93' },
  { value: 'revising', label: '修改中', color: '#f5a623' },
  { value: 'final', label: '定稿', color: '#34c759' },
]


watch(() => props.modelValue, (v) => {
  if (v && props.doc) {
    selectedIcon.value = props.doc.icon || ''
    selectedStatus.value = props.doc.doc_status || 'draft'
    wordGoal.value = props.doc.word_goal || 0
    title.value = props.doc.title || ''
  }
})

function close() {
  emit('update:modelValue', false)
}

function save() {
  const trimmed = title.value.trim()
  const data: any = {
    icon: selectedIcon.value,
    // 章节允许空名（自动「第N章」编号）；其他类型空则保留原名
    title: trimmed || (props.doc?.type === 'chapter' ? '' : (props.doc?.title || '')),
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
/* 基础信息明细行 */
.meta-list {
  margin-bottom: 16px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 10px;
  padding: 4px 12px;
}
.meta-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 6px 0;
  font-size: 13px;
  white-space: nowrap;
}
.meta-item + .meta-item { border-top: 1px dashed var(--border, #e5e5ea); }
.meta-k { color: var(--text-secondary, #86868b); }
.meta-v { color: var(--text-primary, #1d1d1f); font-variant-numeric: tabular-nums; }

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
.stat-val.sm { font-size: 13px; font-weight: 500; }
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
