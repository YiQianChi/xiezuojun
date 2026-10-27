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

      <!-- 章节不显示序号——标题自带「第X章」，数字是冗余噪音 -->
      <span v-if="node.type !== 'chapter'" class="node-index">{{ index }}</span>
      <span class="node-icon" :class="{ 'char-avatar': node.type === 'character' }">{{ nodeIcon }}</span>
      <span v-if="charRole" class="char-role-tag" :style="{ background: charRole.bg }">{{ charRole.label }}</span>
      <span
        v-if="!isFolder"
        class="status-dot"
        :style="{ background: statusColor }"
        :title="statusLabel"
        @click.stop="emit('cycle-status', node)"
      ></span>
      <span class="node-title" :class="titleSizeClass">{{ displayTitle }}</span>
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
import { chapterDisplayTitle } from '../core/chapterTitle'

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

// ponytail: 设定区平铺后只有卷是容器；characters（人物）是第二个容器——角色卡（character 类型）挂它下面
const isFolder = computed(() => props.node.type === 'volume' || props.node.type === 'characters')

const nodeIcon = computed(() => {
  // ponytail: 用户自选图标优先，否则按 type 默认
  if (props.node.icon) return props.node.icon
  // 角色卡：meta.avatar（首字或 emoji）优先
  if (props.node.type === 'character') return charAvatar.value
  const map: Record<string, string> = {
    volume: '📁',
    folder: '📂',
    outline: '📋',
    characters: '👥',
    worldbuilding: '🌍',
    storyline: '📖',
    chapter_outline: '📑',
    chapter: '📄',
    note: '📄',
  }
  return map[props.node.type] || '📄'
})

// 角色卡 meta（无则空对象）
const charMeta = computed<Record<string, any>>(() => {
  if (props.node.type !== 'character' || !props.node.meta) return {}
  try { return JSON.parse(props.node.meta) } catch { return {} }
})
const charAvatar = computed(() => charMeta.value.avatar || '🧑')
// 角色徽标（主/配/反/路人）
const ROLE_STYLES: Record<string, { label: string; bg: string }> = {
  main: { label: '主', bg: '#81bb8e' },
  support: { label: '配', bg: '#8fa9c9' },
  villain: { label: '反', bg: '#d98a7a' },
  extra: { label: '众', bg: '#b3bfb5' },
}
const charRole = computed(() => ROLE_STYLES[charMeta.value.role] || null)

// ponytail: 章节标题 = 自动序号 + 作者写的名字（位置变序号自动跟随）
const displayTitle = computed(() =>
  props.node.type === 'chapter' ? chapterDisplayTitle(props.node, props.allDocs) : props.node.title
)

// 长标题自动缩字号，尽量完整展示（Boss：以美观为主，能显示全就别省略号）
// 侧栏可用宽度约 240px，去掉图标/状态点/缩进后标题区 ~175px：12px≈14字、10.5px≈16字
const titleSizeClass = computed(() => {
  const len = [...String(displayTitle.value || '')].length
  if (len > 14) return 'title-shrink-2'
  if (len > 11) return 'title-shrink-1'
  return ''
})

// ponytail: 文件夹标签按 type 推导
const childLabel = computed(() => {
  const t = props.node.type
  if (t === 'volume') return '章节'
  if (t === 'characters') return '角色'
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
/* 角色卡头像（首字/emoji）：品牌绿渐变小方块 */
.node-icon.char-avatar {
  width: 20px; height: 20px; border-radius: 6px;
  background: linear-gradient(135deg, #a8ceb2, #7fb28d);
  color: #fff; font-size: 11px; font-weight: 700;
  display: inline-flex; align-items: center; justify-content: center;
  opacity: 1;
}
.char-role-tag {
  font-size: 9px; color: #fff; border-radius: 4px; padding: 0 4px;
  flex-shrink: 0; line-height: 14px;
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

/* 长标题缩字号尽量展示全 */
.node-title.title-shrink-1 { font-size: 12px; }
.node-title.title-shrink-2 { font-size: 10.5px; letter-spacing: -0.3px; }

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