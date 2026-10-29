<template>
  <aside class="doc-tree">

    <!-- 今日码字卡（数据来自编辑器实时统计：todayWords=本次会话净增，chapterWords=当前章） -->
    <div class="goal-card">
      <div class="row">
        <span class="k">今日码字</span>
        <span class="v">{{ todayWords || 0 }}<small> 字</small></span>
      </div>
      <div class="bar"><i :style="{ width: Math.min(100, (todayWords || 0) / 20) + '%' }"></i></div>
      <div class="meta"><span>本章 {{ chapterWords || 0 }} 字</span><span>今日刻度 2000 字</span></div>
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

    <!-- 方案B（2026-10-29）：左栏双 Tab——默认正文，作品相关收进第二 Tab -->
    <div class="side-tabs">
      <div class="side-tab" :class="{ on: sideTab === 'chapters' }" @click="sideTab = 'chapters'">
        正文<span class="cnt">{{ articleRootNodes.length }}</span>
      </div>
      <div class="side-tab" :class="{ on: sideTab === 'related' }" @click="sideTab = 'related'">
        作品相关<span class="cnt">{{ settingRootNodes.length }}</span>
      </div>
    </div>

    <div class="tree-list" @contextmenu.prevent="$emit('context-menu', $event, null)">
      <div v-if="loadingDocs" class="tree-loading">加载中...</div>
      <div v-else-if="settingRootNodes.length === 0 && articleRootNodes.length === 0" class="tree-empty">
        还没有文档<br>
        <button @click="$emit('init-default')" class="btn small">初始化默认结构</button>
      </div>
      <template v-else>
        <!-- Tab：作品相关 -->
        <div v-show="sideTab === 'related' || loadingDocs" class="setting-zone">
          <div class="zone-caption">作品相关</div>
          <DocTreeNode
            v-for="(node, i) in settingRootNodes"
            :key="node.id"
            :node="node"
            :all-docs="documents"
            :active-id="activeDocId"
            :depth="0"
            :index="i + 1"
            compact
            @select="$emit('select-doc', $event)"
            @context-menu="(e, node) => $emit('context-menu', e, node)"
            @drop-node="(src, dst, pos) => $emit('drop-node', src, dst, pos)"
            @add-child="(folder) => $emit('add-doc-in', folder)"
            @cycle-status="(n) => $emit('cycle-status', n)"
          />
          <div class="zone-add-link" @click.stop="$emit('add-setting')">＋ 新建设定</div>
        </div>

        <!-- 分隔线（双 Tab 后仅作品相关 tab 内展示） -->
        <div v-if="sideTab === 'related' && settingRootNodes.length > 0 && articleRootNodes.length > 0" class="zone-divider"></div>

        <!-- Tab：正文 -->
        <div v-show="sideTab === 'chapters' || loadingDocs" class="article-zone">
          <div class="zone-caption">正文</div>
          <DocTreeNode
            v-for="(node, i) in articleRootNodes"
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
          <div class="zone-add-link article" @click.stop="$emit('add-article')">＋ 新建卷</div>
        </div>
      </template>
    </div>

    <!-- 书卡：当前作品信息 -->
    <div class="book-card">
      <div class="cover">{{ (projectName || '书')[0] }}</div>
      <div class="bc-main">
        <div class="bt">{{ projectName || '未命名作品' }}</div>
        <div class="st">{{ articleRootNodes.length }} 卷 · {{ documents.length }} 篇</div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DocTreeNode from '../DocTreeNode.vue'

const props = defineProps<{
  branches: any[]
  activeBranchId: number | null
  documents: any[]
  settingRootNodes: any[]
  articleRootNodes: any[]
  activeDocId: number | null
  loadingDocs: boolean
  projectName?: string
  todayWords?: number
  chapterWords?: number
}>()

/* 方案B：左栏双 Tab（默认正文） */
const sideTab = ref<'chapters' | 'related'>('chapters')

defineEmits<{
  'switch-branch': [id: number]
  'init-default': []
  'select-doc': [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
  'add-doc-root': []
  'add-setting': []
  'add-article': []
  'add-doc-in': [folder: any]
  'cycle-status': [node: any]
}>()</script>

<style scoped>
/* ===== 方案B：左栏双 Tab ===== */
.side-tabs {
  display: flex;
  gap: 4px;
  margin: 8px 12px 6px;
}
.side-tab {
  flex: 1;
  text-align: center;
  font-size: 13px;
  padding: 7px 0;
  border-radius: 9px;
  cursor: pointer;
  color: var(--text-secondary);
  background: rgba(0, 0, 0, 0.045);
  transition: all 0.15s;
  user-select: none;
}
.side-tab:hover:not(.on) { color: var(--text-primary); }
.side-tab.on {
  background: var(--bg-secondary, #fbf7ee);
  color: var(--text-primary);
  font-weight: 600;
  box-shadow: 0 1px 4px rgba(96, 74, 40, 0.10);
  border: 1px solid var(--setting-border, #e6dcc6);
}
.side-tab .cnt {
  font-size: 10px;
  opacity: 0.65;
  margin-left: 3px;
}

/* ===== 今日码字卡（demo：纸张布局 v1） ===== */
.goal-card {
  margin: 12px 12px 4px;
  padding: 12px 14px;
  border-radius: 12px;
  background: #fbf7ee;
  border: 1px solid #e6dcc6;
  box-shadow: 0 1px 4px rgba(96, 74, 40, 0.06);
}
.goal-card .row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.goal-card .k { font-size: 12px; color: #94897a; }
.goal-card .v {
  font-size: 22px;
  font-weight: 700;
  color: #33302a;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}
.goal-card .v small { font-size: 12px; font-weight: 400; color: #94897a; }
.goal-card .bar {
  height: 5px;
  border-radius: 3px;
  background: #ece3d0;
  margin-top: 8px;
  overflow: hidden;
}
.goal-card .bar i {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #8fc59b, #6aa877);
  border-radius: 3px;
  transition: width 0.5s ease;
}
.goal-card .meta {
  display: flex;
  justify-content: space-between;
  margin-top: 6px;
  font-size: 11px;
  color: #94897a;
}

/* ===== 书卡（demo：纸张布局 v1） ===== */
.book-card {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 10px 14px 12px;
  border-top: 1px solid #e2d8c2;
  flex-shrink: 0;
}
.book-card .cover {
  width: 34px;
  height: 46px;
  border-radius: 4px;
  flex-shrink: 0;
  background: linear-gradient(160deg, #8fc59b, #6aa877);
  color: #fff;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 3px rgba(96, 74, 40, 0.18);
}
.book-card .bc-main { min-width: 0; }
.book-card .bt {
  font-size: 13px;
  font-weight: 600;
  color: #33302a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.book-card .st { font-size: 11px; color: #94897a; margin-top: 3px; }

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

/* ===== 设定/文章分区 ===== */
.setting-zone {
  background: var(--setting-bg);
  border-radius: 8px;
  padding: 6px 4px 2px;
  margin-bottom: 4px;
  border: 1px solid var(--setting-border);
}

.zone-add-link {
  padding: 3px 8px;
  margin: 2px 4px 0;
  font-size: 10px;
  color: var(--setting-title);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s ease;
}
.zone-add-link:hover {
  color: var(--accent, #007aff);
  background: rgba(0,122,255,0.06);
}
.zone-add-link.article {
  color: var(--text-tertiary);
  margin-top: 4px;
}

.zone-caption {
  font-size: 10px;
  color: var(--text-tertiary);
  padding: 2px 8px 4px;
  letter-spacing: 1px;
  user-select: none;
}
.zone-divider {
  height: 1px;
  background: var(--border);
  margin: 4px 8px;
}
.article-zone {
  padding-top: 2px;
}
</style>
