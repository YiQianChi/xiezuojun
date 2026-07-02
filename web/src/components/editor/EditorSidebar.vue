<template>
  <aside class="doc-tree">
    <div class="side-tabs">
      <button
        class="side-tab"
        :class="{ active: sideTab === 'chapter' }"
        @click="$emit('update:sideTab', 'chapter')"
      >章节</button>
      <button
        class="side-tab"
        :class="{ active: sideTab === 'setting' }"
        @click="$emit('update:sideTab', 'setting')"
      >设定</button>
    </div>

    <div class="branch-switcher">
      <label>故事线</label>
      <select
        :value="activeBranchId"
        @change="$emit('switch-branch', Number(($event.target as HTMLSelectElement).value))"
        :disabled="branches.length <= 1"
      >
        <option v-for="branch in branches" :key="branch.id" :value="branch.id">
          {{ branch.name }}
        </option>
      </select>
    </div>

    <div class="tree-list" @contextmenu.prevent="$emit('context-menu', $event, null)">
      <div v-if="loadingDocs" class="tree-loading">加载中...</div>
      <div v-else-if="filteredRootNodes.length === 0" class="tree-empty">
        <template v-if="sideTab === 'chapter'">
          还没有章节<br>
          <button @click="$emit('init-default')" class="btn small">初始化默认结构</button>
        </template>
        <template v-else>
          还没有设定<br>
          <span class="tree-empty-hint">大纲 / 人物 / 世界观 会显示在这里</span>
        </template>
      </div>
      <DocTreeNode
        v-else
        v-for="node in filteredRootNodes"
        :key="node.id"
        :node="node"
        :all-docs="documents"
        :active-id="activeDocId"
        :depth="0"
        @select="$emit('select-doc', $event)"
        @context-menu="(e, node) => $emit('context-menu', e, node)"
        @drop-node="(src, dst, pos) => $emit('drop-node', src, dst, pos)"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import DocTreeNode from '../DocTreeNode.vue'

defineProps<{
  sideTab: 'chapter' | 'setting'
  branches: any[]
  activeBranchId: number | null
  documents: any[]
  filteredRootNodes: any[]
  activeDocId: number | null
  loadingDocs: boolean
}>()

defineEmits<{
  'update:sideTab': [tab: 'chapter' | 'setting']
  'switch-branch': [id: number]
  'init-default': []
  'select-doc': [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
}>()
</script>

<style scoped>
.doc-tree {
  width: 260px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-secondary);
  flex-shrink: 0;
}

.side-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  margin: 10px 12px 6px;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
}
.side-tab {
  flex: 1;
  border: none;
  background: transparent;
  padding: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.side-tab:hover:not(.active) { color: var(--text-primary); }
.side-tab.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  font-weight: 500;
}

.branch-switcher {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-primary);
}
.branch-switcher label {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}
.branch-switcher select {
  flex: 1;
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 12px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  outline: none;
}

.tree-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 6px 20px;
}
.tree-loading, .tree-empty {
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
  padding: 20px 12px;
  line-height: 1.7;
}
.tree-empty-hint {
  display: block;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-tertiary);
}

.btn.small {
  padding: 4px 10px;
  font-size: 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
  margin-top: 6px;
}
</style>
