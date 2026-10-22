<template>
  <div class="ds-trigger" :class="{ open }">
    <button class="memory-btn" @click.stop="open = !open" title="显示设置（字号/行距/字距/底部留白）">
      <span v-if="!open">⚙️</span>
      <span v-else>✕</span>
    </button>
    <div class="ds-popup" v-show="open" @click.stop>
      <div class="ds-title">显示设置</div>

      <div class="ds-row">
        <span class="ds-label">字号</span>
        <input type="range" min="14" max="22" step="0.5"
          :value="val('fontSize')" @input="set('fontSize', $event)" />
        <span class="ds-value">{{ val('fontSize') }}px</span>
      </div>

      <div class="ds-row">
        <span class="ds-label">行距</span>
        <input type="range" min="1.5" max="2.4" step="0.05"
          :value="val('lineHeight')" @input="set('lineHeight', $event)" />
        <span class="ds-value">{{ val('lineHeight').toFixed(2) }}</span>
      </div>

      <div class="ds-row">
        <span class="ds-label">字间距</span>
        <input type="range" min="0" max="0.1" step="0.005"
          :value="val('letterSpacing')" @input="set('letterSpacing', $event)" />
        <span class="ds-value">{{ val('letterSpacing').toFixed(2) }}em</span>
      </div>

      <div class="ds-row">
        <span class="ds-label">底部留白</span>
        <input type="range" min="20" max="200" step="10"
          :value="val('bottomPad')" @input="set('bottomPad', $event)" />
        <span class="ds-value">{{ val('bottomPad') }}px</span>
      </div>

      <button class="ds-reset" @click="reset">恢复默认</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

// 只把「用户改过的项」透传给编辑器（CSS 变量），没改的不下发——
// 这样专注模式/移动端的默认排版行为完全不受影响
const props = defineProps<{ overrides: Record<string, number> }>()
const emit = defineEmits<{ update: [key: string, value: number | undefined] }>()

const open = ref(false)

// 展示用默认值（与 editor-page.css 的各端默认值一致）
const isMobile = typeof window !== 'undefined' && window.matchMedia('(max-width: 768px)').matches
const DEFAULTS: Record<string, number> = isMobile
  ? { fontSize: 16, lineHeight: 1.95, letterSpacing: 0, bottomPad: 96 }
  : { fontSize: 17, lineHeight: 1.9, letterSpacing: 0, bottomPad: 60 }

function val(key: string): number {
  return props.overrides[key] ?? DEFAULTS[key]
}

function set(key: string, e: Event) {
  emit('update', key, Number((e.target as HTMLInputElement).value))
}

function reset() {
  for (const k of Object.keys(DEFAULTS)) emit('update', k, undefined)
  open.value = false
}
</script>

<style scoped>
.ds-trigger { position: relative; display: inline-flex; }
.memory-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  padding: 2px 6px;
  color: var(--text-secondary);
  border-radius: 6px;
}
.memory-btn:hover { background: var(--bg-tertiary, #f5f5f7); color: var(--text-primary); }
.ds-popup {
  position: absolute;
  bottom: 34px;
  right: 0;
  width: 260px;
  background: var(--bg-secondary, #fff);
  border-radius: 12px;
  box-shadow: 0 8px 30px rgba(0,0,0,.15);
  padding: 14px 16px;
  z-index: 150;
}
.ds-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 10px;
}
.ds-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
}
.ds-label {
  font-size: 12px;
  color: var(--text-secondary);
  width: 52px;
  flex-shrink: 0;
}
.ds-row input[type=range] {
  flex: 1;
  accent-color: var(--accent, #007aff);
  min-width: 0;
}
.ds-value {
  font-size: 11px;
  color: var(--text-tertiary);
  width: 44px;
  text-align: right;
  flex-shrink: 0;
}
.ds-reset {
  width: 100%;
  margin-top: 10px;
  padding: 6px 0;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: transparent;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}
.ds-reset:hover { color: var(--text-primary); background: var(--bg-tertiary, #f5f5f7); }
@media (max-width: 720px) {
  .ds-popup { position: fixed; left: 16px; right: 16px; bottom: 64px; width: auto; }
}
</style>
