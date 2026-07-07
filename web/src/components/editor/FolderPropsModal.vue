<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="modal" @click.stop>
      <h3>文件夹属性</h3>
      <div class="prop-row">
        <label class="prop-label">显示在</label>
        <div class="tab-checks">
          <label class="tab-check" :class="{ active: scopes.includes('chapter') }">
            <input type="checkbox" :checked="scopes.includes('chapter')" @change="toggleScope('chapter')" />
            <span>章节</span>
          </label>
          <label class="tab-check" :class="{ active: scopes.includes('setting') }">
            <input type="checkbox" :checked="scopes.includes('setting')" @change="toggleScope('setting')" />
            <span>设定</span>
          </label>
        </div>
        <p class="prop-hint">勾选文件夹显示在哪个标签页下，可多选</p>
      </div>
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
        <p class="prop-hint">选择一个图标便于在文件树中区分</p>
      </div>
      <div class="modal-actions">
        <button class="btn" @click="close">取消</button>
        <button class="btn primary" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  doc: any
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  save: [data: { tab_scope: string; icon: string }]
}>()

const scopes = ref<string[]>([])
const selectedIcon = ref('')

// ponytail: 常用图标选择 — 文件夹/设定/写作/标记/物件
const iconChoices = [
  '📁', '📋', '👥', '🌍', '📖', '📑', '📝', '📄',
  '🗂️', '🏷️', '⭐', '💡', '🔒', '⚔️', '🏰', '🗺️',
  ' ✒️', '✏️', '🖊️', '📚', '🎨', '🎬', '🎵', '💎',
  '🔥', '❄️', '🌙', '☀️', '⚡', '🌟', '🎭', '🧩',
]

watch(() => props.modelValue, (v) => {
  if (v && props.doc) {
    // 解析 tab_scope，旧数据按 type 推导
    const ts = props.doc.tab_scope
    if (ts) {
      scopes.value = ts.split(',').map((s: string) => s.trim()).filter(Boolean)
    } else {
      const t = props.doc.type
      if (t === 'volume' || t === 'chapter') scopes.value = ['chapter']
      else scopes.value = ['setting']
    }
    selectedIcon.value = props.doc.icon || ''
  }
})

function toggleScope(scope: string) {
  const i = scopes.value.indexOf(scope)
  if (i >= 0) scopes.value.splice(i, 1)
  else scopes.value.push(scope)
}

function close() {
  emit('update:modelValue', false)
}

function save() {
  // 至少选一个 tab
  if (scopes.value.length === 0) scopes.value = ['chapter']
  emit('save', {
    tab_scope: scopes.value.join(','),
    icon: selectedIcon.value,
  })
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
  width: 420px;
  max-width: 90vw;
  background: var(--bg-secondary, #fff);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  padding: 24px;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}
h3 {
  margin: 0 0 16px 0;
  font-family: var(--font-serif);
  font-size: 18px;
  color: var(--text-primary, #1d1d1f);
}
.prop-row {
  margin-bottom: 20px;
}
.prop-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary, #1d1d1f);
  margin-bottom: 8px;
}
.prop-hint {
  margin: 6px 0 0 0;
  font-size: 11px;
  color: var(--text-secondary, #86868b);
  line-height: 1.4;
}
.tab-checks {
  display: flex;
  gap: 10px;
}
.tab-check {
  padding: 6px 16px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s;
}
.tab-check.active {
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.08);
  color: var(--accent, #007aff);
}
.tab-check input {
  margin: 0;
  width: 14px;
  height: 14px;
  accent-color: var(--accent, #007aff);
}
.icon-picker {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
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
.icon-chip:hover {
  border-color: var(--accent, #007aff);
}
.icon-chip.active {
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.08);
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.15);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
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
</style>
