<template>
  <aside class="doc-tree">

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
        还没有文档<br>
        <button @click="$emit('init-default')" class="btn small">初始化默认结构</button>
      </div>
      <DocTreeNode
        v-else
        v-for="(node, i) in filteredRootNodes"
        :key="node.id"
        :node="node"
        :all-docs="documents"
        :active-id="activeDocId"
        :depth="0"
        :index="i + 1"
        @select="$emit('select-doc', $event)"
        @context-menu="(e, node) => $emit('context-menu', e, node)"
        @drop-node="(src, dst, pos) => $emit('drop-node', src, dst, pos)"
        @add-child="(folder) => $emit('add-doc-in', folder)"
        @cycle-status="(n) => $emit('cycle-status', n)"
      />
      <button
        v-if="!loadingDocs"
        class="add-item-btn"
        @click="$emit('add-doc-root')"
      >+ 新建文件夹</button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import DocTreeNode from '../DocTreeNode.vue'

const props = defineProps<{
  branches: any[]
  activeBranchId: number | null
  documents: any[]
  filteredRootNodes: any[]
  activeDocId: number | null
  loadingDocs: boolean
}>()

defineEmits<{
  'switch-branch': [id: number]
  'init-default': []
  'select-doc': [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
  'add-doc-root': []
  'add-doc-in': [folder: any]
  'cycle-status': [node: any]
}>()</script>

<style scoped>
.doc-tree {
  width: 240px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-secondary);
  flex-shrink: 0;
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
.add-item-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 6px;
  margin-top: 4px;
  border: 1px dashed var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--text-tertiary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.add-item-btn:hover {
  color: var(--accent, #007aff);
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.04);
}
</style>
