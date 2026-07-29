<template>
  <div class="doc-tree-node" :class="{ folder: isFolder, compact: compact }">
    <div
      class="tree-row"
      :class="{ active: node.id === activeId, 'drop-before': dropPos === 'before', 'drop-after': dropPos === 'after', 'drop-inside': dropPos === 'inside' }"
      :style="{ paddingLeft: depth * 12 + 6 + 'px' }"
      @click="handleClick"
      @contextmenu.prevent.stop="$emit('context-menu', $event, node)"
      draggable="true"
      @dragstart="onDragStart"
      @dragover.prevent="onDragOver"
      @dragleave="onDragLeave"
      @drop.prevent="onDrop"
    >
      <span v-if="isFolder" class="toggle-icon" @click.stop="toggleExpand">
        {{ expanded ? '▼' : '▶' }}
      </span>
      <span v-else class="toggle-icon spacer"></span>

      <span class="node-index">{{ index }}</span>
      <span class="node-icon">{{ nodeIcon }}</span>
      <span
        v-if="!isFolder"
        class="status-dot"
        :style="{ background: statusColor }"
        :title="statusLabel"
        @click.stop="emit('cycle-status', node)"
      ></span>
      <span class="node-title">{{ displayTitle }}</span>
      <div v-if="!isFolder && node.word_goal > 0" class="mini-progress-wrap" @click.stop>
        <div class="mini-progress-bar">
          <div class="mini-progress-fill" :style="{ width: progressPct + '%', background: progressColor }"></div>
        </div>
        <span class="mini-progress-text">{{ node.word_count || 0 }}/{{ node.word_goal }}</span>
      </div>
    </div>

    <div v-if="isFolder && expanded" class="tree-children">
      <DocTreeNode
        v-for="(child, i) in children"
        :key="child.id"
        :node="child"
        :all-docs="allDocs"
        :active-id="activeId"
        :depth="depth + 1"
        :index="i + 1"
        :compact="compact"
        @select="$emit('select', $event)"
        @context-menu="(e, n) => $emit('context-menu', e, n)"
        @drop-node="(srcId, dstId, pos) => $emit('drop-node', srcId, dstId, pos)"
        @add-child="(folder) => $emit('add-child', folder)"
        @cycle-status="(n) => $emit('cycle-status', n)"
      />
      <div class="add-child-btn" @click.stop="$emit('add-child', node)">+ 在「{{ displayTitle }}」下新建{{ childLabel }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  node: any
  allDocs: any[]
  activeId: number | string | null
  depth: number
  index: number
  compact?: boolean
}>()

const emit = defineEmits<{
  select: [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
  'add-child': [node: any]
  'cycle-status': [node: any]
}>()

const dropPos = ref<'before' | 'after' | 'inside' | null>(null)

// ponytail: expanded 状态挂在 node 对象上（响应式），而非组件本地 ref。
// loadDocs → loadingDocs 翻转会卸载/重建整棵 DocTreeNode 树，
// 组件本地 ref 会被重置为 true → 用户手动折叠的节点全部弹开。
const expanded = computed(() => props.node._expanded !== false)

const children = computed(() => {
  return props.allDocs
    .filter(d => d.parent_id === props.node.id)
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})

// ponytail: volume + folder + 设定五大类都是文件夹容器
const isFolder = computed(() => {
  const t = props.node.type
  return t === 'volume' || t === 'folder' || t === 'outline' || t === 'characters' || t === 'worldbuilding' || t === 'storyline' || t === 'chapter_outline'
})

const nodeIcon = computed(() => {
  // ponytail: 用户自选图标优先，否则按 type 默认
  if (props.node.icon) return props.node.icon
  const map: Record<string, string> = {
    volume: '📁',
    folder: '📝',
    outline: '📋',
    characters: '👥',
    worldbuilding: '🌍',
    storyline: '📖',
    chapter_outline: '📑',
    chapter: '📄',
    note: '📝',
  }
  return map[props.node.type] || '📄'
})

// ponytail: 作者自己决定章节标题，不做自动编号
const displayTitle = computed(() => props.node.title)

// ponytail: 文件夹标签按 type 推导
const childLabel = computed(() => {
  const t = props.node.type
  if (t === 'volume') return '章节'
  return '设定'
})

function toggleExpand() {
  props.node._expanded = !expanded.value
}

function handleClick() {
  if (isFolder.value) {
    toggleExpand()
  } else {
    emit('select', props.node)
  }
}

function onDragStart(e: DragEvent) {
  if (e.dataTransfer) {
    e.dataTransfer.setData('text/plain', String(props.node.id))
    e.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const y = e.clientY - rect.top
  const h = rect.height

  if (isFolder.value && y > h * 0.25 && y < h * 0.75) {
    dropPos.value = 'inside'
  } else if (y < h * 0.5) {
    dropPos.value = 'before'
  } else {
    dropPos.value = 'after'
  }
}

function onDragLeave(_e: DragEvent) {
  dropPos.value = null
}

function onDrop(e: DragEvent) {
  const pos = dropPos.value
  dropPos.value = null
  const srcId = Number(e.dataTransfer?.getData('text/plain'))
  if (srcId && srcId !== props.node.id && pos) {
    emit('drop-node', srcId, props.node.id, pos)
  }
}

// B3: 状态圆点
const statusColorMap: Record<string, string> = {
  draft: '#8e8e93',
  revising: '#f5a623',
  final: '#34c759',
}
const statusLabelMap: Record<string, string> = {
  draft: '草稿',
  revising: '修改中',
  final: '定稿',
}
const statusColor = computed(() => statusColorMap[props.node.doc_status || 'draft'] || '#999')
const statusLabel = computed(() => statusLabelMap[props.node.doc_status || 'draft'] || '草稿')

// B6: 迷你进度条
const progressPct = computed(() => {
  const goal = props.node.word_goal || 0
  if (!goal) return 0
  return Math.min(100, Math.round(((props.node.word_count || 0) / goal) * 100))
})
const progressColor = computed(() => {
  const goal = props.node.word_goal || 0
  if (!goal) return '#4caf50'
  const ratio = (props.node.word_count || 0) / goal
  if (ratio > 1) return '#e53935'
  if (ratio >= 1) return '#007AFF'
  return '#4caf50'
})
</script>

<style scoped>
.doc-tree-node {
  user-select: none;
}

.tree-row {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
  transition: all var(--transition-fast);
  gap: 3px;
  border-radius: var(--radius-sm);
  margin: 2px 4px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-sm);
}

.tree-row:hover {
  background: var(--accent-subtle);
  border-color: var(--accent-light);
  transform: translateX(2px);
}

.tree-row.active {
  background: var(--accent-light);
  color: var(--accent);
  font-weight: 600;
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--accent-light);
}

.tree-row.drop-before {
  border-top: 2px solid var(--accent);
  border-radius: var(--radius-xs) var(--radius-xs) var(--radius-sm) var(--radius-sm);
}

.tree-row.drop-after {
  border-bottom: 2px solid var(--accent);
  border-radius: var(--radius-sm) var(--radius-sm) var(--radius-xs) var(--radius-xs);
}

.tree-row.drop-inside {
  background: var(--accent-light);
  border-color: var(--accent);
  border-style: dashed;
}

/* 文件夹节点视觉略深一点 */
.doc-tree-node.folder > .tree-row {
  background: var(--bg-primary);
  font-weight: 500;
}

.toggle-icon {
  display: inline-block;
  width: 10px;
  font-size: 8px;
  color: var(--text-tertiary);
  text-align: center;
  flex-shrink: 0;
}

.toggle-icon.spacer {
  visibility: hidden;
}

.node-icon {
  font-size: 14px;
  opacity: 0.9;
  flex-shrink: 0;
}

.node-index {
  font-size: 10px;
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
  margin-right: -2px;
}

.node-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* B3: 状态圆点 */
.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
  cursor: pointer;
  box-shadow: 0 0 0 2px rgba(0,0,0,0.08);
  transition: transform 0.15s ease;
}
.status-dot:hover {
  transform: scale(1.3);
}

/* B6: 迷你进度条 */
.mini-progress-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.mini-progress-bar {
  width: 40px;
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
}
.mini-progress-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
}
.mini-progress-text {
  font-size: 10px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

.tree-children {
  margin-left: 2px;
  padding-left: 4px;
  border-left: 2px solid var(--border-subtle);
}
.add-child-btn {
  padding: 4px 8px;
  margin: 2px 4px;
  font-size: 11px;
  color: var(--text-tertiary);
  cursor: pointer;
  border: 1px dashed var(--border);
  border-radius: 6px;
  transition: all 0.15s ease;
}
.add-child-btn:hover {
  color: var(--accent, #007aff);
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.04);
}

/* ===== compact 模式：设定区紧凑暖灰样式 ===== */
.compact > .tree-row {
  font-size: 11.5px;
  padding: 3px 6px;
  margin: 1px 2px;
  color: var(--setting-text);
  background: var(--setting-card-bg);
  border: 1px solid var(--setting-border);
  box-shadow: none;
  gap: 3px;
}
.compact > .tree-row:hover {
  background: var(--setting-hover);
  border-color: var(--setting-hover-border);
  transform: none;
}
.compact > .tree-row.active {
  background: rgba(0,122,255,0.08);
  color: var(--accent, #007aff);
  font-weight: 500;
  border-color: rgba(0,122,255,0.3);
  box-shadow: none;
}
.compact > .tree-row .node-icon {
  font-size: 11px;
}
.compact > .tree-row .node-index {
  font-size: 9px;
}
.compact > .tree-row .status-dot {
  width: 8px;
  height: 8px;
  box-shadow: none;
}
.compact > .tree-row .toggle-icon {
  font-size: 6px;
  width: 8px;
}
.compact > .tree-children {
  margin-left: 2px;
  padding-left: 4px;
  border-left-color: var(--setting-children-border);
}
.compact > .tree-children .add-child-btn {
  font-size: 10px;
  padding: 2px 6px;
}
.compact > .tree-row .mini-progress-wrap {
  display: none;
}
</style>