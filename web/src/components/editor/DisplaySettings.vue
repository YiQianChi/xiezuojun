<template>
  <div class="ds-trigger" :class="{ open }">
    <button class="memory-btn" @click.stop="open = !open" title="显示设置（字号/行距/字距/视线位置）">
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
        <span class="ds-label">视线位置</span>
        <input type="range" min="30" max="70" step="5"
          :value="val('eyeLine')" @input="set('eyeLine', $event)" />
        <span class="ds-value">{{ val('eyeLine') }}%</span>
      </div>
      <div class="ds-hint">打字时文字行保持在屏幕的这个高度，眼睛不用追着屏幕底部跑</div>

      <div class="ds-row ds-toggle-row">
        <span class="ds-label">段落行号</span>
        <label class="ds-switch">
          <input type="checkbox" :checked="lineNumbers" @change="toggleLineNumbers($event)" />
          <span class="ds-slider"></span>
        </label>
        <span class="ds-value">{{ lineNumbers ? '开' : '关' }}</span>
      </div>

      <button class="ds-reset" @click="reset">恢复默认</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

// 只把「用户改过的项」透传给编辑器（CSS 变量），没改的不下发——
// 这样专注模式/移动端的默认排版行为完全不受影响
const props = defineProps<{ overrides: Record<string, number>, lineNumbers?: boolean }>()
const emit = defineEmits<{ update: [key: string, value: number | undefined], 'toggle-line-numbers': [on: boolean] }>()

const open = ref(false)

// 段落行号开关（独立于 overrides，布尔型；默认关）
const lineNumbers = computed(() => props.lineNumbers === true)
function toggleLineNumbers(e: Event) {
  emit('toggle-line-numbers', (e.target as HTMLInputElement).checked)
}

// 展示用默认值（与 editor-page.css 的各端默认值一致；视线位置两端一致）
const isMobile = typeof window !== 'undefined' && window.matchMedia('(max-width: 768px)').matches
const DEFAULTS: Record<string, number> = isMobile
  ? { fontSize: 16, lineHeight: 1.95, letterSpacing: 0, eyeLine: 45 }
  : { fontSize: 17, lineHeight: 1.9, letterSpacing: 0, eyeLine: 45 }

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
.ds-hint {
  font-size: 11px;
  color: var(--text-tertiary);
  line-height: 1.5;
  padding: 2px 0 4px;
}
/* 段落行号开关 */
.ds-toggle-row { padding-top: 8px; }
.ds-switch {
  position: relative;
  width: 36px;
  height: 20px;
  flex-shrink: 0;
  cursor: pointer;
}
.ds-switch input {
  opacity: 0;
  width: 0;
  height: 0;
  position: absolute;
}
.ds-slider {
  position: absolute;
  inset: 0;
  background: var(--border, #e0e0e0);
  border-radius: 20px;
  transition: background .15s;
}
.ds-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 2px;
  top: 2px;
  background: #fff;
  border-radius: 50%;
  transition: transform .15s;
  box-shadow: 0 1px 3px rgba(0,0,0,.25);
}
.ds-switch input:checked + .ds-slider {
  background: var(--accent, #007aff);
}
.ds-switch input:checked + .ds-slider::before {
  transform: translateX(16px);
}
@media (max-width: 720px) {
  .ds-popup { position: fixed; left: 16px; right: 16px; bottom: 64px; width: auto; }
}
</style>
