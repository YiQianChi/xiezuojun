<template>
  <div v-if="visible" class="search-panel" @keydown.stop>
    <div class="search-header">
      <div class="scope-tabs">
        <button
          class="scope-tab"
          :class="{ active: scope === 'chapter' }"
          @click="switchScope('chapter')"
          title="Ctrl/Cmd+F"
        >本章</button>
        <button
          class="scope-tab"
          :class="{ active: scope === 'project' }"
          @click="switchScope('project')"
          title="Ctrl/Cmd+Shift+F"
        >全项目</button>
      </div>
      <span class="search-icon" aria-hidden="true">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
      </span>
      <input
        ref="inputRef"
        :value="query"
        @input="onInput"
        class="search-input"
        :placeholder="scope === 'chapter' ? '在本章节中查找...' : '在全项目中查找（≥2字）...'"
        @keydown.enter.prevent="onEnter($event)"
        @keydown.esc.prevent="$emit('close')"
      />
      <span
        v-if="query"
        class="match-badge"
        :class="{ none: !loading && total === 0 }"
      >
        <template v-if="loading">…</template>
        <template v-else-if="total === 0">0/0</template>
        <template v-else>{{ activeIndex + 1 }}/{{ total }}</template>
      </span>
      <button class="icon-btn" @click="$emit('prev')" :disabled="total === 0" title="上一个 (Shift+Enter)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18 15 12 9 6 15"/></svg>
      </button>
      <button class="icon-btn" @click="$emit('next')" :disabled="total === 0" title="下一个 (Enter)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="6 9 12 15 18 9"/></svg>
      </button>
      <button class="icon-btn close-btn" @click="$emit('close')" title="关闭 (Esc)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6L6 18"/><path d="M6 6l12 12"/></svg>
      </button>
    </div>

    <!-- 全项目模式：显示分组结果列表 -->
    <div v-if="scope === 'project' && query" class="project-results">
      <div v-if="loading" class="empty-hint">搜索中...</div>
      <div v-else-if="total === 0" class="empty-hint">没有找到匹配</div>
      <div v-else>
        <div v-for="group in projectGroups" :key="group.docId" class="doc-group">
          <div class="doc-header">
            <span class="doc-title">{{ group.title }}</span>
            <span class="doc-count">{{ group.matches.length }}</span>
          </div>
          <div
            v-for="m in group.matches"
            :key="`${group.docId}-${m.occurrenceIndex}`"
            class="match-item"
            :class="{ active: isActiveMatch(m) }"
            @click="$emit('jumpProject', m)"
          >
            <span class="line-num">{{ m.line }}</span>
            <span class="snippet" v-html="highlight(m.snippet)"></span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'

export interface ProjectMatch {
  docId: number
  title: string
  line: number
  col: number
  snippet: string
  occurrenceIndex: number
}

const props = defineProps<{
  visible: boolean
  scope: 'chapter' | 'project'
  query: string
  total: number
  activeIndex: number
  loading?: boolean
  projectMatches?: ProjectMatch[]
}>()

const emit = defineEmits<{
  close: []
  next: []
  prev: []
  'update:query': [q: string]
  'update:scope': [s: 'chapter' | 'project']
  jumpProject: [m: ProjectMatch]
}>()

const inputRef = ref<HTMLInputElement>()

function onInput(e: Event) {
  const target = e.target as HTMLInputElement
  emit('update:query', target.value)
}

function onEnter(e: KeyboardEvent) {
  if (e.shiftKey) emit('prev')
  else emit('next')
}

function switchScope(s: 'chapter' | 'project') {
  if (s === props.scope) return
  emit('update:scope', s)
  nextTick(() => inputRef.value?.focus())
}

const projectGroups = computed(() => {
  const list = props.projectMatches || []
  const map = new Map<number, { docId: number; title: string; matches: ProjectMatch[] }>()
  for (const m of list) {
    if (!map.has(m.docId)) map.set(m.docId, { docId: m.docId, title: m.title, matches: [] })
    map.get(m.docId)!.matches.push(m)
  }
  return Array.from(map.values())
})

function isActiveMatch(m: ProjectMatch) {
  const list = props.projectMatches || []
  const idx = list.findIndex(x => x.docId === m.docId && x.occurrenceIndex === m.occurrenceIndex)
  return idx === props.activeIndex
}

function highlight(text: string): string {
  const q = props.query.trim()
  if (!q) return escapeHtml(text)
  const safeText = escapeHtml(text)
  const safeQ = escapeRegex(q)
  return safeText.replace(new RegExp(safeQ, 'gi'), (m) => `<mark>${m}</mark>`)
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

watch(() => props.visible, async (v) => {
  if (v) {
    await nextTick()
    inputRef.value?.focus()
    inputRef.value?.select()
  }
})

onMounted(() => {
  if (props.visible) {
    nextTick(() => inputRef.value?.focus())
  }
})

defineExpose({ focus: () => inputRef.value?.focus() })
</script>

<style scoped>
.search-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 380px;
  max-width: calc(100% - 24px);
  max-height: calc(100vh - 140px);
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 0 0 1px rgba(0, 0, 0, 0.05);
  z-index: 450;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}

.search-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 10px;
  flex-wrap: wrap;
}

.scope-tabs {
  display: inline-flex;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  padding: 2px;
  margin-right: 4px;
}
.scope-tab {
  border: none;
  background: transparent;
  color: var(--text-secondary, #86868b);
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.scope-tab:hover {
  color: var(--text-primary, #1d1d1f);
}
.scope-tab.active {
  background: var(--bg-secondary, #ffffff);
  color: var(--accent, #007aff);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  font-weight: 500;
}

.search-icon {
  color: var(--text-secondary, #86868b);
  display: inline-flex;
  padding: 0 2px 0 4px;
}

.search-input {
  flex: 1;
  min-width: 140px;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  padding: 4px 2px;
}

.match-badge {
  font-size: 11px;
  color: var(--text-secondary, #86868b);
  padding: 2px 6px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.04);
  min-width: 40px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
.match-badge.none {
  color: #d0453d;
  background: rgba(208, 69, 61, 0.08);
}

.icon-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary, #86868b);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.icon-btn:hover:not(:disabled) {
  background: var(--bg-tertiary, #f0f0f0);
  color: var(--text-primary, #1d1d1f);
}
.icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.close-btn { margin-left: 2px; }

/* 全项目结果 */
.project-results {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 0 8px;
  border-top: 1px solid var(--border-subtle, #f0f0f0);
}

.doc-group + .doc-group {
  border-top: 1px solid var(--border-subtle, #f0f0f0);
}

.doc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px 4px;
  font-size: 11px;
  color: var(--text-secondary, #86868b);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.doc-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.doc-count {
  background: rgba(0, 0, 0, 0.04);
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
}

.match-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 5px 12px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary, #1d1d1f);
  border-left: 2px solid transparent;
}
.match-item:hover {
  background: rgba(0, 122, 255, 0.04);
}
.match-item.active {
  background: rgba(0, 122, 255, 0.08);
  border-left-color: var(--accent, #007aff);
}

.line-num {
  color: var(--text-tertiary, #aeaeb2);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  min-width: 28px;
  flex-shrink: 0;
}

.snippet {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.6;
}
.snippet :deep(mark) {
  background: rgba(255, 214, 10, 0.55);
  color: inherit;
  border-radius: 3px;
  padding: 0 1px;
}

.empty-hint {
  padding: 20px 12px;
  text-align: center;
  color: var(--text-tertiary, #aeaeb2);
  font-size: 12px;
}
</style>
