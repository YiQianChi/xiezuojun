<template>
  <!-- $attrs 显式绑定：本组件是多根模板（aside + Teleport 新建菜单），Vue 不会自动继承
       fallthrough class——Editor 传的 drawer-open 曾被静默丢弃，导致手机端抽屉打不开 -->
  <aside class="doc-tree" v-bind="$attrs">

    <!-- 今日码字卡（数据来自编辑器实时统计：todayWords=本次会话净增，chapterWords=当前章） -->
    <div class="goal-card">
      <div class="row">
        <span class="k">今日码字</span>
        <span class="v">{{ todayWords || 0 }}<small> / {{ dailyGoal }}</small></span>
      </div>
      <div class="bar"><i :style="{ width: Math.min(100, ((todayWords || 0) / dailyGoal) * 100) + '%' }"></i></div>
      <div class="meta">
        <span>本章 {{ chapterWords || 0 }} 字</span>
        <span v-if="!goalEditing" class="goal-edit" @click="startGoalEdit" title="点此修改每日目标">🎯 目标 {{ dailyGoal }} 字 ✎</span>
        <span v-else class="goal-editing">
          <input
            ref="goalInputRef"
            v-model="goalDraft"
            type="number"
            min="100"
            step="100"
            @keyup.enter="saveGoal"
            @keyup.escape="goalEditing = false"
            @blur="saveGoal"
          />
        </span>
      </div>
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
        <button :disabled="initBusy" @click="$emit('init-default')" class="btn small">{{ initBusy ? '初始化中…' : '初始化默认结构' }}</button>
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
          <div class="zone-add article-wrap">
            <div ref="addMenuAnchor" class="zone-add-link article" @click.stop="toggleAddMenu">＋ 新建 ▾</div>
          </div>
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

  <!-- 新建菜单：Teleport 到 body——原绝对定位弹层长在 .tree-list（overflow 滚动容器）里会被裁剪压住 -->
  <Teleport to="body">
    <div v-if="addMenuOpen" class="add-menu teleported" :style="addMenuStyle" @click.stop>
      <div @click="$emit('add-volume'); addMenuOpen = false">📚 新建分卷</div>
      <div @click="$emit('add-prologue'); addMenuOpen = false">🌙 新建楔子<span class="am-sub">插最前 · 不占章节号</span></div>
      <div @click="$emit('add-extra'); addMenuOpen = false">🎁 新建番外<span class="am-sub">放最后 · 不占章节号</span></div>
    </div>
  </Teleport>
</template>

<script lang="ts">
// 多根模板必须关掉自动继承，配合根元素 v-bind="$attrs"（见模板注释）
export default { inheritAttrs: false }
</script>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import DocTreeNode from '../DocTreeNode.vue'

const props = defineProps<{
  documents: any[]
  settingRootNodes: any[]
  articleRootNodes: any[]
  activeDocId: number | null
  loadingDocs: boolean
  projectName?: string
  todayWords?: number
  chapterWords?: number
  initBusy?: boolean
}>()

/* 方案B：左栏双 Tab（默认正文） */
const sideTab = ref<'chapters' | 'related'>('chapters')
const addMenuOpen = ref(false)
const addMenuAnchor = ref<HTMLElement | null>(null)
const addMenuStyle = ref<Record<string, string>>({})
function toggleAddMenu() {
  if (addMenuOpen.value) { addMenuOpen.value = false; return }
  const el = addMenuAnchor.value
  if (el) {
    const r = el.getBoundingClientRect()
    addMenuStyle.value = { left: `${Math.max(8, r.right - 196)}px`, bottom: `${window.innerHeight - r.top + 6}px` }
  }
  addMenuOpen.value = true
}
// 点别处收起菜单（teleport 到 body 后必须自己管）；capture 阶段要放过触发钮/菜单自身，否则切换会被「先关再开」抵消
function onDocClickCloseMenu(e: MouseEvent) {
  const t = e.target as HTMLElement | null
  if (t?.closest && (t.closest('.add-menu') || t.closest('.zone-add-link'))) return
  addMenuOpen.value = false
}
if (typeof document !== 'undefined') {
  document.addEventListener('click', onDocClickCloseMenu, { capture: true })
}

/* 每日目标可调：全局 localStorage，默认 2000 */
const GOAL_KEY = 'xzj-daily-goal'
const dailyGoal = ref(2000)
const goalEditing = ref(false)
const goalDraft = ref('2000')
const goalInputRef = ref<HTMLInputElement | null>(null)

try {
  const saved = Number(localStorage.getItem(GOAL_KEY))
  if (saved >= 100) dailyGoal.value = saved
} catch { /* ignore */ }

function startGoalEdit() {
  goalDraft.value = String(dailyGoal.value)
  goalEditing.value = true
  nextTick(() => goalInputRef.value?.focus())
}

function saveGoal() {
  if (!goalEditing.value) return
  goalEditing.value = false
  const n = Math.round(Number(goalDraft.value))
  if (n >= 100 && n <= 50000 && n !== dailyGoal.value) {
    dailyGoal.value = n
    try { localStorage.setItem(GOAL_KEY, String(n)) } catch { /* ignore */ }
  }
}

defineEmits<{
  'init-default': []
  'select-doc': [node: any]
  'context-menu': [e: MouseEvent, node: any]
  'drop-node': [srcId: number, dstId: number, position: 'before' | 'after' | 'inside']
  'add-doc-root': []
  'add-setting': []
  'add-volume': []
  'add-prologue': []
  'add-extra': []
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
.goal-card .meta .goal-edit {
  cursor: pointer;
  transition: color 0.15s;
}
.goal-card .meta .goal-edit:hover { color: #4d8a5c; }
.goal-card .meta .goal-editing input {
  width: 64px;
  border: 1px solid #8fc59b;
  border-radius: 5px;
  font-size: 11px;
  padding: 1px 5px;
  outline: none;
  background: #fff;
  color: #33302a;
  text-align: right;
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

/* ＋ 新建下拉菜单（对齐炼笔智作：分卷/楔子/番外） */
.article-wrap { position: relative; }
.add-menu {
  position: absolute;
  right: 6px;
  bottom: calc(100% + 4px);
  z-index: 260;
  min-width: 190px;
  background: #fffdf8;
  border: 1px solid #e2d8c2;
  border-radius: 10px;
  box-shadow: 0 8px 28px rgba(60, 45, 20, 0.16);
  padding: 5px;
}
/* Teleport 到 body 的版本：视口定位，压过一切滚动容器/层叠上下文 */
.add-menu.teleported {
  position: fixed;
  right: auto;
  bottom: auto;
  z-index: 900;
}
.add-menu > div {
  padding: 8px 12px;
  font-size: 12.5px;
  border-radius: 7px;
  cursor: pointer;
  color: var(--text-primary);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}
.add-menu > div:hover { background: rgba(129, 187, 142, 0.14); color: #4d8a5c; }
.add-menu .am-sub { font-size: 10px; color: var(--text-tertiary); }

/* 深色适配 v1.3：Teleport 新建菜单暖底 → 暗面 */
html.dark .add-menu {
  background: #2c2c2e;
  border-color: #ffffff1a;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.6);
}

</style>
