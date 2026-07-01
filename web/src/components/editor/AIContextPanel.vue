<template>
  <div class="context-panel" :class="{ collapsed: !open }">
    <div
      class="context-panel-toggle"
      @click="$emit('update:open', !open)"
      :title="open ? '收起' : '展开'"
    >
      <svg v-if="open" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="9 18 15 12 9 6"/></svg>
      <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="15 18 9 12 15 6"/></svg>
    </div>

    <div class="context-panel-inner" v-show="open">
      <div class="context-header">
        <h4>AI 参考设定</h4>
        <label class="toggle-inline" title="关掉后 AI 只看前文，不参考人物/大纲/世界观">
          <input
            type="checkbox"
            :checked="includeSettings"
            @change="$emit('update:includeSettings', ($event.target as HTMLInputElement).checked)"
          />
          <span>启用</span>
        </label>
      </div>

      <div class="context-body" v-if="!includeSettings">
        <div class="context-empty">
          <div class="ctx-icon">🚫</div>
          <div>AI 不参考项目设定<br><span class="hint">开启后，续写/改写会自动读取大纲/人物/世界观</span></div>
        </div>
      </div>

      <div class="context-body" v-else-if="usedSettings.length === 0">
        <div class="context-empty">
          <div class="ctx-icon">💡</div>
          <div>还没生成过<br><span class="hint">下一次 AI 续写/改写时，这里会显示实际参考的设定</span></div>
        </div>
      </div>

      <div class="context-body" v-else>
        <div class="context-stats">
          本次参考 {{ usedCount }} / {{ usedSettings.length }} 张 · {{ totalTokens }} tokens
        </div>
        <div
          v-for="s in usedSettings"
          :key="s.id"
          class="context-chip"
          :class="[s.type, { unused: !s.used }]"
          @click="$emit('jump', s.id)"
          :title="s.used ? '点击跳转到该设定' : '因空间不足未参与，点击跳转查看'"
        >
          <div class="chip-head">
            <span class="chip-type">{{ typeLabel(s.type) }}</span>
            <span class="chip-title">{{ s.title }}</span>
            <span v-if="s.used" class="chip-tokens">{{ s.tokens }}t</span>
            <span v-else class="chip-unused">未用</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface UsedSetting {
  id: number
  title: string
  type: 'outline' | 'characters' | 'worldbuilding'
  used: boolean
  tokens: number
}

const props = defineProps<{
  open: boolean
  includeSettings: boolean
  usedSettings: UsedSetting[]
}>()

defineEmits<{
  'update:open': [v: boolean]
  'update:includeSettings': [v: boolean]
  jump: [docId: number]
}>()

const usedCount = computed(() => props.usedSettings.filter(s => s.used).length)
const totalTokens = computed(() => props.usedSettings.reduce((sum, s) => sum + (s.tokens || 0), 0))

function typeLabel(t: string): string {
  return { outline: '大纲', characters: '人物', worldbuilding: '世界观' }[t] || t
}
</script>

<style scoped>
.context-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  bottom: 12px;
  width: 280px;
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06), 0 0 0 1px rgba(0, 0, 0, 0.04);
  z-index: 300;
  display: flex;
  flex-direction: row;
  overflow: hidden;
  transition: width 0.2s ease;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}
.context-panel.collapsed {
  width: 24px;
  background: transparent;
  box-shadow: none;
}
.context-panel-toggle {
  width: 24px;
  min-width: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-secondary, #86868b);
  background: var(--bg-tertiary, #f0f0f0);
  border-radius: 6px 0 0 6px;
  transition: background 0.15s;
}
.context-panel.collapsed .context-panel-toggle {
  background: var(--bg-secondary, #ffffff);
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1), 0 0 0 1px rgba(0, 0, 0, 0.05);
}
.context-panel-toggle:hover { background: rgba(0, 0, 0, 0.08); color: var(--text-primary, #1d1d1f); }

.context-panel-inner {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.context-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 10px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}
.context-header h4 {
  margin: 0;
  font-size: 12px;
  color: var(--text-tertiary, #aeaeb2);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}
.toggle-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-secondary, #86868b);
  cursor: pointer;
  user-select: none;
}
.toggle-inline input[type="checkbox"] {
  accent-color: var(--accent, #007aff);
  width: 12px; height: 12px;
  margin: 0;
}

.context-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px 12px;
}
.context-empty {
  padding: 30px 12px;
  text-align: center;
  color: var(--text-tertiary, #aeaeb2);
  font-size: 12px;
  line-height: 1.6;
}
.context-empty .ctx-icon { font-size: 24px; margin-bottom: 8px; opacity: 0.6; }
.context-empty .hint { color: var(--text-tertiary, #aeaeb2); opacity: 0.8; font-size: 11px; }

.context-stats {
  font-size: 11px;
  color: var(--text-tertiary, #aeaeb2);
  padding: 4px 4px 8px;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.05);
  margin-bottom: 6px;
}

.context-chip {
  background: rgba(0, 122, 255, 0.05);
  border-radius: 8px;
  padding: 8px 10px;
  margin-bottom: 6px;
  cursor: pointer;
  transition: all 0.15s;
  border-left: 3px solid transparent;
}
.context-chip:hover { background: rgba(0, 122, 255, 0.1); }
.context-chip.outline { border-left-color: #ff9500; }
.context-chip.characters { border-left-color: #af52de; }
.context-chip.worldbuilding { border-left-color: #34c759; }
.context-chip.unused { opacity: 0.5; background: rgba(0, 0, 0, 0.03); }

.chip-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}
.chip-type {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.06);
  color: var(--text-secondary, #86868b);
  flex-shrink: 0;
}
.context-chip.outline .chip-type { background: rgba(255, 149, 0, 0.15); color: #c96f00; }
.context-chip.characters .chip-type { background: rgba(175, 82, 222, 0.15); color: #8a3fb0; }
.context-chip.worldbuilding .chip-type { background: rgba(52, 199, 89, 0.15); color: #268a3f; }

.chip-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary, #1d1d1f);
  font-weight: 500;
}
.chip-tokens {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.chip-unused {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  flex-shrink: 0;
}
</style>
