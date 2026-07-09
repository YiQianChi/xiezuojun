<template>
      <div class="editor-page" :class="{ 'focus-mode': focusMode }">
    <!-- 顶部栏 -->
    <EditorHeader
      :project-name="projectName"
      :current-doc-name="currentDocName"
      :mode="mode"
      :active-doc-id="activeDocId"
      :saving-version="savingVersion"
      :focus-mode="focusMode"
      @back="goBack"
      @switch-mode="switchMode"
      @save-version="openSaveVersionModal"
      @open-history="openVersionModal"
      @toggle-focus="focusMode = !focusMode"
    />

    <div class="editor-body">
      <!-- 左侧文档树 -->
      <EditorSidebar
        :side-tab="sideTab"
        :branches="branches"
        :active-branch-id="activeBranchId"
        :documents="documents"
        :filtered-root-nodes="filteredRootNodes"
        :active-doc-id="activeDocId"
        :loading-docs="loadingDocs"
        @update:side-tab="sideTab = $event"
        @switch-branch="switchBranch"
        @init-default="initDefaultStructure"
        @select-doc="openDoc"
        @context-menu="openDocContextMenu"
        @drop-node="handleDropNode"
        @add-doc-root="createDocAtRoot"
        @add-doc-in="createDocInFolder"
      />

      <!-- 编辑器 -->
      <div class="editor-main" :class="{ 'ai-panel-open': contextPanelOpen }" v-show="mode === 'write'">
        <div v-if="compareActive" class="compare-view">
          <div class="compare-topbar">
            <div>
              <strong>对比当前版本</strong>
              <span>{{ selectedVersion?.label || '历史版本' }}</span>
            </div>
            <div class="compare-actions">
              <label class="sync-toggle">
                <span>同步滚动</span>
                <input v-model="compareSyncScroll" type="checkbox" />
                <i></i>
              </label>
              <button class="btn" @click="closeCompareView">退出对比</button>
              <button class="btn primary" @click="applyCompareCurrent">保存当前修改</button>
            </div>
          </div>
          <div class="compare-panes">
            <div class="compare-pane old">
              <div class="pane-title">历史版本（只读）</div>
              <div class="diff-lines" ref="compareOldRef" @scroll="syncCompareScroll('old')">
                <div
                  v-for="(row, index) in compareRows"
                  :key="'old-' + index"
                  class="diff-line"
                  :class="row.type"
                >
                  {{ row.oldText }}
                </div>
              </div>
            </div>
            <div class="compare-pane current">
              <div class="pane-title">当前版本（可修改）</div>
              <textarea ref="compareCurrentRef" v-model="compareCurrentContent" @input="onCompareCurrentInput" @scroll="syncCompareScroll('current')"></textarea>
            </div>
          </div>
        </div>
        <div v-else-if="!activeDocId" class="editor-empty">
          <p>👈 请选择或新建一个章节</p>
        </div>
        <div v-else class="editor-wrapper" @contextmenu.prevent="openContextMenu">
          <div class="tiptap-container">
            <editor-content v-if="editor" :editor="editor" />
          </div>
        </div>
        <SearchPanel
          :visible="searchVisible"
          :scope="searchScope"
          :query="searchQuery"
          :total="searchTotal"
          :active-index="searchActiveIndex"
          :loading="searchLoading"
          :project-matches="searchProjectMatches"
          @close="closeSearch"
          @next="searchNext"
          @prev="searchPrev"
          @update:query="onSearchQueryUpdate"
          @update:scope="onSearchScopeChange"
          @jump-project="onJumpProject"
        />

        <!-- AI 参考面板（写作模式） -->
        <AIContextPanel
          :open="contextPanelOpen"
          :project-id="projectId"
          :used-settings="usedSettings"
          @update:open="contextPanelOpen = $event"
          @jump="jumpToSetting"
        />
      </div>

      <!-- 图谱浮层（编辑器内右下角） -->
      <div class="graph-fab">
        <button class="graph-fab-btn" @click="graphOpen = !graphOpen" :title="graphOpen ? '收起图谱' : '关系图谱'">
          <span v-if="!graphOpen">🔗</span>
          <span v-else>✕</span>
        </button>
        <div class="graph-fab-panel" :class="{ maximized: graphMaximized }" v-show="graphOpen" @click.stop>
          <button class="graph-resize-btn" @click="graphMaximized = !graphMaximized" :title="graphMaximized ? '缩小' : '放大'">
            <svg v-if="!graphMaximized" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M16 3h3a2 2 0 0 1 2 2v3"/><path d="M8 21H5a2 2 0 0 1-2-2v-3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
          </button>
          <RelationshipGraph
            :project-id="String(projectId)"
            :chapters="chapterList"
            :current-chapter="currentChapterIndex"
          />
        </div>
      </div>

      <!-- 计划模式 -->
      <PlanView v-if="mode === 'plan'" :project-name="projectName" :project-id="projectId" @jump-to-doc="jumpToSetting" />

      <!-- 审核模式 -->
      <ReviewView v-if="mode === 'review'" :project-id="projectId" @jump-to-source="handleReviewJump" @ai-fix="handleReviewAiFix" />

      <!-- AI 选区工具栏（选择时） -->
      <div
        v-if="aiToolbar.visible"
        class="ai-toolbar"
        :style="{ left: aiToolbar.x + 'px', top: aiToolbar.y + 'px' }"
      >
        <button @mousedown.prevent @click="prepareAIAction('continue')">续写</button>
        <button @mousedown.prevent @click="prepareAIAction('rewrite')">改写</button>
        <button @mousedown.prevent @click="prepareAIAction('polish')">润色</button>
        <button @mousedown.prevent @click="prepareAIAction('expand')">扩写</button>
      </div>

      <!-- AI 底部固定栏（方向输入 / 生成中 / 结果预览） -->
      <div v-if="aiPhase !== 'idle'" class="ai-bottom-bar" :class="'phase-' + aiPhase">
        <!-- 方向输入 -->
        <template v-if="aiPhase === 'inputting'">
          <button class="ai-icon-btn" @click="cancelInputting" aria-label="取消">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6L6 18"/></svg>
          </button>
          <div class="ai-input-wrap">
            <input
              ref="aiDirectionInputRef"
              v-model="aiDirection"
              class="ai-direction-input"
              type="text"
              :placeholder="pendingAction === 'continue' ? '续写方向（可选）' : pendingAction === 'rewrite' ? '改写方向（可选）' : pendingAction === 'polish' ? '润色方向（可选）' : '扩写方向（可选）'"
              @keydown.enter.prevent="executeAIAction"
              @keydown.escape.prevent="cancelInputting"
            />
          </div>
          <button class="ai-icon-btn primary" @click="executeAIAction" aria-label="生成">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linejoin="round"><polygon points="6 4 20 12 6 20 6 4" fill="currentColor"/></svg>
          </button>
        </template>
        <!-- 生成中 -->
        <template v-if="aiPhase === 'generating'">
          <span class="ai-bottom-hint">生成中…</span>
          <button class="ai-icon-btn danger" @click="stopAIGeneration" aria-label="停止">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><rect x="5" y="5" width="14" height="14" rx="2"/></svg>
          </button>
        </template>
        <!-- 结果预览 -->
        <template v-if="aiPhase === 'preview'">
          <span class="ai-bottom-hint">已生成</span>
          <button class="ai-icon-btn" @click="regenerateAIResult" :disabled="aiLoading" aria-label="重新生成">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 11-3-6.7L21 8M21 3v5h-5"/></svg>
          </button>
          <button class="ai-icon-btn danger" @click="discardAIResult" aria-label="丢弃">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/></svg>
          </button>
          <button class="ai-icon-btn success" @click="acceptAIResult" aria-label="采用">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          </button>
        </template>
      </div>

      <!-- 编辑器右键菜单 -->
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      >
        <div class="ctx-highlight-row">
          <span
            v-for="c in highlightColors"
            :key="c"
            class="hl-color-dot"
          :style="{ background: c }"
          @click="contextAction('highlight', c)"
          ></span>
        </div>
        <div class="ctx-sep"></div>
        <div @click="contextAction('copy')">复制</div>
        <div @click="contextAction('cut')">剪切</div>
        <div @click="contextAction('paste')">粘贴</div>
      </div>

      <!-- 文档树右键菜单 -->
      <div
        v-if="docContextMenu.visible"
        class="context-menu"
        :style="{ left: docContextMenu.x + 'px', top: docContextMenu.y + 'px' }"
      >
        <template v-if="docContextMenu.doc">
          <div @click="renameDoc">重命名</div>
          <div v-if="isFolderNode(docContextMenu.doc)" @click="openFolderProps">属性</div>
          <div @click="deleteDoc" class="danger">删除</div>
        </template>
        <template v-else>
          <div @click="newDocUnder('chapter')">新建文件</div>
          <div @click="newDocUnder('volume')">新建文件夹</div>
        </template>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="status-bar">
      <span class="status-item">{{ wordCount }} 字</span>
      <span class="status-item">{{ saveStatus }}</span>
      <div class="memory-trigger" :class="{ open: memoryOpen }">
        <button class="memory-btn" @click="memoryOpen = !memoryOpen" :title="memoryOpen ? '收起记忆' : '记忆'">
          <span v-if="!memoryOpen">🧠</span>
          <span v-else>✕</span>
        </button>
        <div class="memory-popup" v-show="memoryOpen" @click.stop>
          <MemoryPanel :project-id="String(projectId)" />
        </div>
      </div>
      <div class="memory-trigger" :class="{ open: threadOpen }">
        <button class="memory-btn" @click="threadOpen = !threadOpen" :title="threadOpen ? '收起追踪' : '伏笔追踪'">
          <span v-if="!threadOpen">🧵</span>
          <span v-else>✕</span>
        </button>
        <div class="memory-popup thread-popup" v-show="threadOpen" @click.stop>
          <ThreadPanel :project-id="String(projectId)" />
        </div>
      </div>
    </div>

    <!-- 新建文档弹窗 -->
    <CreateDocModal
      v-model="showCreateDocModal"
      :doc-type="createDocType"
      :side-tab="sideTab"
      :folder-type-options="createDocFolderTypeOptions"
      @create="onCreateDocConfirm"
    />

    <!-- 文件夹属性弹窗 -->
    <FolderPropsModal
      v-model="showFolderPropsModal"
      :doc="folderPropsDoc"
      @save="onFolderPropsSave"
    />

    <!-- 保存版本名称弹窗 -->
    <SaveVersionModal
      v-model="showSaveVersionModal"
      :saving="savingVersion"
      @save="onSaveVersionConfirm"
    />

    <!-- 版本历史弹窗 -->
    <VersionHistoryModal
      v-model="showVersionModal"
      :loading="loadingVersions"
      :restoring="restoringVersion"
      :versions="versions"
      :selected-version="selectedVersion"
      @select="selectVersion"
      @compare="startCompareWithCurrent"
      @restore="restoreSelectedVersion"
    />

    <!-- 专注模式退出提示 -->
    <div class="focus-exit-hint" @click="focusMode = false">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
      <span>退出专注 (Esc)</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Highlight from '@tiptap/extension-highlight'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
// ponytail: 4 fixed highlight colors, no config needed
const highlightColors = ['#ffeb3b', '#4caf50', '#2196f3', '#ff5252']
import { documentApi, branchApi, projectApi } from '../core/api'
import SearchPanel from '../components/SearchPanel.vue'
import PlanView from '../components/PlanView.vue'
import ReviewView from '../components/ReviewView.vue'
import EditorHeader from '../components/editor/EditorHeader.vue'
import EditorSidebar from '../components/editor/EditorSidebar.vue'
import AIContextPanel from '../components/editor/AIContextPanel.vue'
import MemoryPanel from '../components/editor/MemoryPanel.vue'
import RelationshipGraph from '../components/editor/RelationshipGraph.vue'
import ThreadPanel from '../components/editor/ThreadPanel.vue'
import CreateDocModal from '../components/editor/CreateDocModal.vue'
import FolderPropsModal from '../components/editor/FolderPropsModal.vue'
import SaveVersionModal from '../components/editor/SaveVersionModal.vue'
import VersionHistoryModal from '../components/editor/VersionHistoryModal.vue'
import { useEditorSearch, createSearchHighlightPlugin } from '../composables/useEditorSearch'
import { useAIToolbar, AIMark, AIStrikethrough } from '../composables/useAIToolbar'
import { useVersionManager } from '../composables/useVersionManager'
import { toast } from '../composables/useToast'

const route = useRoute()
const router = useRouter()

const projectId = Number(route.params.projectId)

const editor = shallowRef<any>(null)
const documents = ref<any[]>([])
const branches = ref<any[]>([])
const activeBranchId = ref<number | null>(null)
const activeDocId = ref<number | null>(null)
const currentDocName = ref('')
const projectName = ref('')
const loadingDocs = ref(false)
const wordCount = ref(0)
const saveStatus = ref('')

const contextMenu = ref({ visible: false, x: 0, y: 0 })
// ponytail: 右键时编辑器会失焦丢选区，打开菜单前缓存
let cachedSelection: { text: string; start: number; end: number } | null = null
const docContextMenu = ref({ visible: false, x: 0, y: 0, doc: null as any })
const showCreateDocModal = ref(false)
const newDocName = ref('')
const createDocType = ref<string>('chapter')
const createDocParent = ref<number | null>(null)
const createDocFolderTypeOptions = computed(() => {
  if (createDocType.value !== 'volume') return undefined
  return [
    { value: 'volume', label: '📁 卷/文件夹' },
    { value: 'outline', label: '📋 大纲' },
    { value: 'characters', label: '👥 人物' },
    { value: 'worldbuilding', label: '🌍 世界观' },
    { value: 'storyline', label: '📖 故事主线' },
    { value: 'chapter_outline', label: '📑 章节大纲' },
  ]
})

// 文件夹属性弹窗
const showFolderPropsModal = ref(false)
const folderPropsDoc = ref<any>(null)

let saveTimer: any = null
const isSettingContent = ref(false)

// 版本管理 composable
const {
  // 版本状态
  versions, selectedVersion, loadingVersions, savingVersion, restoringVersion,
  showVersionModal, showSaveVersionModal,
  // 版本操作
  openSaveVersionModal, onSaveVersionConfirm, openVersionModal,
  selectVersion, restoreSelectedVersion,
  // 对比状态
  compareActive, compareCurrentContent, compareSyncScroll,
  compareRows, compareOldRef, compareCurrentRef,
  // 对比操作
  startCompareWithCurrent, closeCompareView, onCompareCurrentInput, syncCompareScroll, applyCompareCurrent,
} = useVersionManager({
  editor, activeDocId, isSettingContent, wordCount, saveStatus,
  autoSave, loadDocs, getEditorText, editorContentFromText,
  hideAIToolbar: () => { aiToolbar.value.visible = false },
  cancelSaveTimer: () => { if (saveTimer) { clearTimeout(saveTimer); saveTimer = null } },
})
// template refs — used via ref="compareOldRef" / ref="compareCurrentRef" in <template>
void compareOldRef; void compareCurrentRef
let selectionHandler: (() => void) | null = null

// 搜索面板状态 & 逻辑（本章 / 全项目 两种模式）— composable
const {
  searchVisible,
  searchScope,
  searchQuery,
  searchTotal,
  searchActiveIndex,
  searchLoading,
  searchProjectMatches,
  openSearch,
  closeSearch,
  onSearchQueryUpdate,
  onSearchScopeChange,
  searchNext,
  searchPrev,
  onJumpProject,
  applyEditorSearch,
} = useEditorSearch({
  editor,
  projectId,
  activeDocId,
  documents,
  openDoc: (...args) => openDoc(...args),
})

// 根节点（parent_id 为空的）
const rootNodes = computed(() => {
  return documents.value
    .filter(d => !d.parent_id)
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})

// 关系图时间轴：正文目录下的章节文档（type=volume/chapter），按 sort_order 展平为线性列表
const CHAPTER_TYPES = ['volume', 'chapter']
const chapterList = computed(() => {
  return documents.value
    .filter(d => CHAPTER_TYPES.includes(d.type))
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})
const currentChapterIndex = computed(() => {
  if (!activeDocId.value) return 0
  const idx = chapterList.value.findIndex(d => d.id === activeDocId.value)
  return idx >= 0 ? idx + 1 : 0
})

// 左侧栏 Tab：章节 / 设定
const sideTab = ref<'chapter' | 'setting'>('chapter')
const memoryOpen = ref(false)
const graphOpen = ref(false)
const threadOpen = ref(false)
const graphMaximized = ref(false)

// 顶部工作模式：计划 / 写作 / 审核
type EditorMode = 'plan' | 'write' | 'review'
const mode = ref<EditorMode>('write')

// AI 参考的设定文档（从后端返回）
interface UsedSetting {
  id: number
  title: string
  type: 'outline' | 'characters' | 'worldbuilding'
  used: boolean
  tokens: number
}
const usedSettings = ref<UsedSetting[]>([])
const contextPanelOpen = ref(false) // 右侧参考面板默认收起，AI 操作时自动弹出
const focusMode = ref(false) // 专注模式：隐藏侧栏+顶栏，编辑器全屏

// AI 工具栏 & 交互 —— composable
const {
  aiLoading,
  aiPhase,
  aiDirection,
  aiToolbar,
  aiInsert,
  showAIToolbar,
  pendingAction,
  prepareAIAction,
  executeAIAction,
  stopAIGeneration,
  acceptAIResult,
  discardAIResult,
  regenerateAIResult,
  cancelInputting,
  quickRewrite,
} = useAIToolbar({
  editor,
  projectId,
  activeDocId,
  usedSettings,
  saveStatus,
})

const aiDirectionInputRef = ref<HTMLInputElement | null>(null)
// ponytail: focus direction input when entering inputting phase
watch(aiPhase, (v) => {
  if (v === 'inputting') {
    contextPanelOpen.value = true // 使用 AI 时自动弹出右侧面板
    nextTick(() => aiDirectionInputRef.value?.focus())
  }
})

async function jumpToSetting(docId: number) {
  let doc = documents.value.find(d => d.id === docId)
  if (!doc) {
    await loadDocs()
    doc = documents.value.find(d => d.id === docId)
  }
  if (doc) {
    sideTab.value = 'setting'
    mode.value = 'write'
    await openDoc(doc, { keepSearch: true })
  }
}

// ponytail: locus text search is fuzzy — matches by doc title substring, good enough for chapter names
function findDocByLocus(locus: string) {
  return documents.value.find(d =>
    d.title === locus || locus.includes(d.title) || d.title.includes(locus)
  )
}

async function handleReviewJump({ locus, desc }: { locus: string; desc: string }) {
  const doc = findDocByLocus(locus)
  if (!doc) return
  switchMode('write')
  await openDoc(doc)
  // highlight and scroll to the problematic text
  await nextTick()
  if (!editor.value) return
  const snippet = desc.length > 30 ? desc.slice(0, 30) : desc
  const fullText = editor.value.state.doc.textBetween(0, editor.value.state.doc.content.size, '\n')
  const pos = fullText.indexOf(snippet)
  if (pos < 0) return
  editor.value.chain()
    .setTextSelection({ from: pos, to: pos + snippet.length })
    .run()
  // scroll selection into view
  const coords = editor.value.view.coordsAtPos(pos)
  const selEl = document.elementFromPoint(coords.left, coords.top)?.parentElement
  if (selEl) selEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

async function handleReviewAiFix(issue: any) {
  const doc = findDocByLocus(issue.locus)
  if (!doc) return
  switchMode('write')
  await openDoc(doc)
  await nextTick()
  if (!editor.value) return
  // select the problematic text snippet in editor
  const snippet = issue.desc?.length > 30 ? issue.desc.slice(0, 30) : issue.desc || ''
  const fullText = editor.value.state.doc.textBetween(0, editor.value.state.doc.content.size, '\n')
  const pos = fullText.indexOf(snippet)
  if (pos >= 0) {
    editor.value.chain()
      .focus()
      .setTextSelection({ from: pos, to: pos + snippet.length })
      .run()
  }
  await quickRewrite(snippet, pos >= 0 ? pos : 0, pos >= 0 ? pos + snippet.length : 0)
}

function switchMode(next: EditorMode) {
  if (mode.value === next) return
  mode.value = next
  // 模式联动侧栏：计划 → 设定；写作/审核 → 章节
  if (next === 'plan') sideTab.value = 'setting'
  else sideTab.value = 'chapter'
  // 关闭浮层，避免残留
  if (aiPhase.value === 'generating') stopAIGeneration()
  aiToolbar.value.visible = false
  aiInsert.value.active = false
  contextMenu.value.visible = false
  if (searchVisible.value) closeSearch()
}

// ponytail: 旧数据无 tab_scope 时按 type 推导
function docTabScope(doc: any): string[] {
  if (doc.tab_scope) return doc.tab_scope.split(',').map((s: string) => s.trim()).filter(Boolean)
  // 兼容旧数据：按 type 推导
  const t = doc.type
  if (t === 'volume' || t === 'chapter') return ['chapter']
  if (['outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline', 'note'].includes(t)) return ['setting']
  return ['chapter']
}

const filteredRootNodes = computed(() => {
  const tab = sideTab.value
  return rootNodes.value.filter(n => docTabScope(n).includes(tab))
})

// 沿 parent_id 向上找到根节点
function findRootDoc(doc: any): any | null {
  if (!doc) return null
  let cur = doc
  const seen = new Set<number>()
  while (cur && cur.parent_id) {
    if (seen.has(cur.id)) break
    seen.add(cur.id)
    const parent = documents.value.find(d => d.id === cur.parent_id)
    if (!parent) break
    cur = parent
  }
  return cur
}

onMounted(async () => {
  initEditor()
  window.addEventListener('keydown', handleGlobalKeydown)
  loadProjectName()
  await loadBranches()
  await loadDocs()
})

async function loadProjectName() {
  try {
    const res = await projectApi.get(projectId)
    const data = res.data?.data || res.data
    projectName.value = data?.name || ''
  } catch (e) {
    console.warn('加载项目名失败', e)
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  if (editor.value) editor.value.destroy()
  if (saveTimer) clearTimeout(saveTimer)
  if (selectionHandler) {
    document.removeEventListener('selectionchange', selectionHandler)
    selectionHandler = null
  }
})

function initEditor() {
  if (editor.value) editor.value.destroy()
  if (selectionHandler) {
    document.removeEventListener('selectionchange', selectionHandler)
  }

  editor.value = new Editor({
    content: '',
    extensions: [
      StarterKit,
      Highlight.configure({ multicolor: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      AIMark,
      AIStrikethrough,
    ],
    // ponytail: 粘贴强制纯文本，去掉外部加粗/斜体等格式
    editorProps: {
      handlePaste: (view, event) => {
        const text = event.clipboardData?.getData('text/plain')
        if (text) {
          view.dispatch(view.state.tr.insertText(text))
          return true
        }
        return false
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (isSettingContent.value) return
      const plainText = getEditorText(ed)
      wordCount.value = plainText.length
      saveStatus.value = '编辑中...'
      if (saveTimer) clearTimeout(saveTimer)
      saveTimer = setTimeout(() => autoSave(), 1500)
    }
  })

  // 注册搜索高亮 ProseMirror 插件
  editor.value.registerPlugin(createSearchHighlightPlugin())

  selectionHandler = () => {
    if (!editor.value || !editor.value.isFocused) return
    if (isSettingContent.value) return
    // AI 活动期间不响应选区变化，防止清空预览
    if (aiPhase.value !== 'idle') return
    const sel = getSelection()
    if (sel.text.trim()) {
      showAIToolbar(sel)
    } else {
      aiToolbar.value.visible = false
    }
  }
  document.addEventListener('selectionchange', selectionHandler)
}

function getSelection() {
  if (!editor.value) return { text: '', start: 0, end: 0 }
  let { from, to } = editor.value.state.selection
  const text = editor.value.state.doc.textBetween(from, to, '\n')
  // ponytail: ProseMirror selection can miss the leading char at text-node boundaries.
  // Cross-check with native selection length; if off by one at the start, correct it.
  const native = window.getSelection()
  if (native && native.rangeCount > 0 && native.toString().length > text.length) {
    const before = from > 0 ? editor.value.state.doc.textBetween(from - 1, from, '\n') : ''
    if (before) from = from - 1
  }
  const correctedText = editor.value.state.doc.textBetween(from, to, '\n')
  return { text: correctedText, start: from, end: to }
}

// ponytail: parse Markdown table blocks (| col | col |) into ProseMirror table nodes
function parseMarkdownTable(lines: string[], startIdx: number): { tableNode: object; consumed: number } | null {
  const headerLine = lines[startIdx]
  // need at least header + separator + 1 row
  if (startIdx + 2 >= lines.length) return null
  const separatorLine = lines[startIdx + 1]
  // separator must be like |---|---| or | --- | :---: |
  if (!/^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)+\|?\s*$/.test(separatorLine)) return null

  // ponytail: simpler split — strip leading/trailing | then split
  const splitCells = (line: string) => {
    let s = line.trim()
    if (s.startsWith('|')) s = s.slice(1)
    if (s.endsWith('|')) s = s.slice(0, -1)
    return s.split('|').map(c => c.trim())
  }

  const headers = splitCells(headerLine)
  if (headers.length < 1) return null

  // ponytail: parse **bold** and *italic* inline marks into ProseMirror text nodes
  const makeTextContent = (text: string) => {
    if (!text) return []
    const nodes: object[] = []
    // split on **bold** or *italic* patterns
    const regex = /(\*\*[^*]+\*\*|\*[^*]+\*)/g
    let last = 0
    let m: RegExpExecArray | null
    while ((m = regex.exec(text)) !== null) {
      if (m.index > last) nodes.push({ type: 'text', text: text.slice(last, m.index) })
      const matched = m[0]
      if (matched.startsWith('**')) {
        nodes.push({ type: 'text', text: matched.slice(2, -2), marks: [{ type: 'bold' }] })
      } else {
        nodes.push({ type: 'text', text: matched.slice(1, -1), marks: [{ type: 'italic' }] })
      }
      last = m.index + matched.length
    }
    if (last < text.length) nodes.push({ type: 'text', text: text.slice(last) })
    return nodes
  }

  const rows: object[] = []
  // header row
  rows.push({
    type: 'tableRow',
    content: headers.map(h => ({ type: 'tableHeader', content: [{ type: 'paragraph', content: makeTextContent(h) }] }))
  })

  // body rows
  let consumed = 2 // header + separator
  for (let i = startIdx + 2; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('|')) break
    const cells = splitCells(line)
    // pad/truncate to header length
    while (cells.length < headers.length) cells.push('')
    rows.push({
      type: 'tableRow',
      content: cells.slice(0, headers.length).map((c: string) => ({ type: 'tableCell', content: [{ type: 'paragraph', content: makeTextContent(c) }] }))
    })
    consumed++
  }

  if (rows.length < 2) return null // need at least header + 1 body row

  return { tableNode: { type: 'table', content: rows }, consumed }
}

function editorContentFromText(content: string) {
  if (!content) return { type: 'doc', content: [{ type: 'paragraph' }] }
  const lines = content.split('\n')
  const blocks: object[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    // detect markdown table: line has | and next line is separator
    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?\s*:?-{2,}:?\s*(\|)/.test(lines[i + 1])) {
      const result = parseMarkdownTable(lines, i)
      if (result) {
        blocks.push(result.tableNode)
        i += result.consumed
        continue
      }
    }
    blocks.push({
      type: 'paragraph',
      content: line ? [{ type: 'text', text: line }] : []
    })
    i++
  }
  return { type: 'doc', content: blocks }
}

function getEditorText(ed = editor.value) {
  if (!ed) return ''
  const blocks: string[] = []
  ed.state.doc.forEach((node: any) => {
    blocks.push(node.textContent || '')
  })
  return blocks.join('\n')
}

// ponytail: save/load as HTML to preserve marks (highlight etc.)
function getEditorHTML(ed = editor.value) {
  return ed ? ed.getHTML() : ''
}

async function loadBranches() {
  try {
    const res = await branchApi.list(projectId)
    branches.value = res.data?.data || res.data || []
    if (!activeBranchId.value && branches.value.length > 0) {
      activeBranchId.value = branches.value[0].id
    }
  } catch (e) {
    console.error('加载故事线失败', e)
    branches.value = []
  }
}

async function switchBranch() {
  compareActive.value = false
  activeDocId.value = null
  currentDocName.value = ''
  wordCount.value = 0
  saveStatus.value = ''
  if (editor.value) editor.value.commands.setContent('', { emitUpdate: false })
  await loadDocs()
}

async function loadDocs() {
  loadingDocs.value = true
  try {
    const res = await documentApi.list(String(projectId), activeBranchId.value)
    documents.value = res.data?.data || res.data || []
  } catch (e) {
    console.error('加载文档失败', e)
    documents.value = []
  } finally {
    loadingDocs.value = false
  }
}

async function initDefaultStructure() {
  const defaults = [
    { type: 'outline', title: '大纲', sort_order: 1 },
    { type: 'characters', title: '人物设定', sort_order: 2 },
    { type: 'worldbuilding', title: '世界观', sort_order: 3 },
    { type: 'volume', title: '正文', sort_order: 4 },
  ]
  for (const d of defaults) {
    try {
      await documentApi.create({
        project_id: projectId,
        title: d.title,
        type: d.type,
        sort_order: d.sort_order,
      })
    } catch (e) {
      console.error('创建默认结构失败', d, e)
    }
  }
  await loadDocs()
}

async function openDoc(doc: any, opts?: { keepSearch?: boolean }) {
  compareActive.value = false
  activeDocId.value = doc.id
  currentDocName.value = doc.title
  aiToolbar.value.visible = false
  // 打开文档时自动跟随其所属分组切 Tab
  const rootDoc = findRootDoc(doc)
  const scopes = rootDoc ? docTabScope(rootDoc) : []
  if (scopes.includes(sideTab.value)) {
    // 当前 Tab 已匹配，不切
  } else if (scopes.length > 0) {
    sideTab.value = scopes[0] as 'chapter' | 'setting'
  }
  // 切换章节时清空搜索状态与高亮（全项目跳转除外）
  if (!opts?.keepSearch && (searchQuery.value || searchTotal.value)) {
    searchQuery.value = ''
    searchTotal.value = 0
    searchActiveIndex.value = 0
    searchProjectMatches.value = []
    applyEditorSearch('', -1)
  }

  try {
    const res = await documentApi.get(doc.id)
    const docData = res.data?.data || res.data
    let content = docData?.content
    if (typeof content !== 'string') content = ''

    if (editor.value) {
      isSettingContent.value = true
      // ponytail: load HTML directly to preserve marks; fallback to plain text for old docs
      const html = content.trim().startsWith('<') ? content : null
      if (html) {
        editor.value.commands.setContent(html, { emitUpdate: false })
      } else {
        editor.value.commands.setContent(editorContentFromText(content), { emitUpdate: false })
      }
      wordCount.value = getEditorText().length
      saveStatus.value = '已保存'
      setTimeout(() => { isSettingContent.value = false }, 100)
    }
  } catch (e) {
    console.error('加载文档内容失败', e)
    isSettingContent.value = false
  }
}

async function autoSave() {
  if (!activeDocId.value || !editor.value) return
  try {
    saveStatus.value = '保存中...'
    await documentApi.update(activeDocId.value, {
      content: getEditorHTML()
    })
    saveStatus.value = '已保存'
  } catch (e) {
    saveStatus.value = '保存失败'
  }
}

function openCreateDocModal(parentId: number | null = null, type: string = 'chapter') {
  createDocParent.value = parentId
  createDocType.value = type
  newDocName.value = ''
  showCreateDocModal.value = true
}

// 拆分组件 CreateDocModal 触发（emit create）
async function onCreateDocConfirm(name: string, folderType?: string) {
  newDocName.value = name
  await createDoc(folderType)
}

async function createDoc(folderType?: string) {
  if (!newDocName.value.trim()) return
  try {
    // 计算 sort_order
    const siblings = documents.value.filter(d =>
      (createDocParent.value === null ? !d.parent_id : d.parent_id === createDocParent.value)
    )
    const maxSort = siblings.reduce((m, d) => Math.max(m, d.sort_order || 0), 0)

    // ponytail: 新建文件夹时 folderType 决定真实 type 和 tab_scope
    const actualType = folderType || createDocType.value
    const isFolderType = ['volume', 'outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline'].includes(actualType)
    const tabScope = isFolderType ? sideTab.value : (actualType === 'chapter' ? 'chapter' : 'setting')

    const res = await documentApi.create({
      project_id: projectId,
      branch_id: activeBranchId.value || undefined,
      title: newDocName.value.trim(),
      type: actualType,
      parent_id: createDocParent.value || undefined,
      sort_order: maxSort + 1,
      tab_scope: tabScope,
    })
    showCreateDocModal.value = false
    newDocName.value = ''
    await loadDocs()
    const newDoc = res.data?.data || res.data
    if (newDoc && actualType === 'chapter') {
      openDoc(newDoc)
    }
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  }
}

function openDocContextMenu(e: MouseEvent, doc: any) {
  docContextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    doc
  }
  document.addEventListener('click', closeDocContextMenu, { once: true })
}

function closeDocContextMenu() {
  docContextMenu.value.visible = false
}

// ponytail: 底部按钮固定根目录新建
function createDocAtRoot() {
  openCreateDocModal(null, 'volume')
}

// ponytail: 文件夹内+按钮
function createDocInFolder(folder: any) {
  openCreateDocModal(folder.id, 'chapter')
}

function newDocUnder(type: 'volume' | 'chapter') {
  const doc = docContextMenu.value.doc
  // ponytail: 设定 tab 新建文件用 note 类型，章节 tab 用 chapter
  const isSetting = sideTab.value === 'setting'
  const actualType = isSetting && type === 'chapter' ? 'note' : type
  // 空白区域右键 → 根级创建；右键文件夹 → 在其下创建；右键文件 → 同级创建
  if (!doc) {
    openCreateDocModal(null, actualType)
  } else {
    const isFolder = isFolderNode(doc)
    const parentId = isFolder ? doc.id : (doc.parent_id || null)
    openCreateDocModal(parentId, actualType)
  }
  closeDocContextMenu()
}

// ponytail: 判断是否文件夹容器（含 tab_scope 多选归属的）
function isFolderNode(doc: any): boolean {
  const t = doc.type
  return t === 'volume' || t === 'outline' || t === 'characters' || t === 'worldbuilding' || t === 'storyline' || t === 'chapter_outline'
}

// 右键文件夹 → 属性
function openFolderProps() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  folderPropsDoc.value = doc
  showFolderPropsModal.value = true
  closeDocContextMenu()
}

async function onFolderPropsSave(data: { tab_scope: string; icon: string }) {
  const doc = folderPropsDoc.value
  if (!doc) return
  try {
    await documentApi.update(doc.id, { tab_scope: data.tab_scope, icon: data.icon })
    showFolderPropsModal.value = false
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '更新属性失败')
  }
}

async function renameDoc() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  closeDocContextMenu()
  const newName = prompt('新名称', doc.title)
  if (!newName || newName === doc.title) return
  try {
    await documentApi.update(doc.id, { title: newName })
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '重命名失败')
  }
}

async function deleteDoc() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  closeDocContextMenu()
  if (!confirm(`确定删除「${doc.title}」？`)) return
  try {
    await documentApi.delete(doc.id)
    if (activeDocId.value === doc.id) {
      activeDocId.value = null
      currentDocName.value = ''
      if (editor.value) editor.value.commands.setContent('')
    }
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '删除失败')
  }
}

async function handleDropNode(srcId: number, dstId: number, position: 'before' | 'after' | 'inside') {
  const src = documents.value.find(d => d.id === srcId)
  const dst = documents.value.find(d => d.id === dstId)
  if (!src || !dst) return

  let newParentId: number | null
  let newSortOrder: number

  if (position === 'inside') {
    // 拖到文件夹中间 → 成为该文件夹的子节点
    newParentId = dst.id
    const children = documents.value.filter(d => d.parent_id === dst.id)
    newSortOrder = children.reduce((m, d) => Math.max(m, d.sort_order || 0), 0) + 1
  } else {
    // before / after → 成为该节点的同级
    newParentId = dst.parent_id || null
    const baseSort = dst.sort_order || 0
    newSortOrder = position === 'before' ? baseSort - 1 : baseSort + 1
  }

  try {
    await documentApi.move(srcId, {
      parent_id: newParentId !== null ? String(newParentId) : null,
      sort_order: newSortOrder,
    })
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '移动失败')
  }
}

function openContextMenu(e: MouseEvent) {
  // ponytail: 右键前缓存当前选区，右键动作后编辑器已失焦
  cachedSelection = getSelection()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY }
  document.addEventListener('click', closeContextMenu, { once: true })
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function contextAction(action: string, color?: string) {
  if (!editor.value) return
  const sel = cachedSelection || getSelection()
  cachedSelection = null
  if (!sel.text && action !== 'paste') return
  switch (action) {
    case 'highlight':
      editor.value.chain().setTextSelection({ from: sel.start, to: sel.end }).toggleHighlight({ color }).run()
      break
    case 'copy':
      navigator.clipboard.writeText(sel.text)
      break
    case 'cut':
      navigator.clipboard.writeText(sel.text)
      editor.value.commands.deleteRange({ from: sel.start, to: sel.end })
      break
    case 'paste':
      navigator.clipboard.readText().then(text => {
        editor.value?.commands.insertContentAt(sel.start, text)
      })
      break
  }
  contextMenu.value.visible = false
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    autoSave()
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
    e.preventDefault()
    openSearch(e.shiftKey ? 'project' : 'chapter')
    return
  }
  if (e.key === 'Escape' && searchVisible.value) {
    e.preventDefault()
    closeSearch()
    return
  }
  // ponytail: Esc 退出专注模式（搜索关闭后优先级最低）
  if (e.key === 'Escape' && focusMode.value) {
    e.preventDefault()
    focusMode.value = false
    return
  }
  // Ctrl/Cmd+Shift+Z 切换专注模式
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === 'z') {
    e.preventDefault()
    focusMode.value = !focusMode.value
    return
  }
}

function goBack() {
  router.push('/bookshelf')
}
</script>

<style scoped src="./editor-page.css"></style>
