<template>
  <div class="doc-tree-node" :class="{ folder: isFolder }">
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
      <span class="node-title">{{ displayTitle }}</span>
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
        @select="$emit('select', $event)"
        @context-menu="(e, n) => $emit('context-menu', e, n)"
        @drop-node="(srcId, dstId, pos) => $emit('drop-node', srcId, dstId, pos)"
        @add-child="(folder) => $emit('add-child', folder)"
      />
      <div class="add-child-btn" @click.stop="$emit('add-child', node)">+ 在「{{ displayTitle }}」下新建章节</div>
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
}>()

const emit = defineEmits<{
  select: [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
  'add-child': [node: any]
}>()

const expanded = ref(true)
const dropPos = ref<'before' | 'after' | 'inside' | null>(null)

const children = computed(() => {
  return props.allDocs
    .filter(d => d.parent_id === props.node.id)
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})

// ponytail: volume + 设定三大类都是文件夹容器
const isFolder = computed(() => {
  const t = props.node.type
  return t === 'volume' || t === 'outline' || t === 'characters' || t === 'worldbuilding'
})

const nodeIcon = computed(() => {
  const map: Record<string, string> = {
    volume: '📁',
    outline: '📋',
    characters: '👥',
    worldbuilding: '🌍',
    chapter: '📄',
    note: '📝',
  }
  return map[props.node.type] || '📄'
})

// ponytail: 作者自己决定章节标题，不做自动编号
const displayTitle = computed(() => props.node.title)

function toggleExpand() {
  expanded.value = !expanded.value
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
</style>