<template>
  <div class="editor-page">
    <!-- 顶部栏 -->
    <EditorHeader
      :project-name="projectName"
      :current-doc-name="currentDocName"
      :mode="mode"
      :active-doc-id="activeDocId"
      :saving-version="savingVersion"
      @back="goBack"
      @switch-mode="switchMode"
      @save-version="openSaveVersionModal"
      @open-history="openVersionModal"
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
      />

      <!-- 编辑器 -->
      <div class="editor-main" v-show="mode === 'write'">
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
          :include-settings="includeSettings"
          :used-settings="usedSettings"
          @update:open="contextPanelOpen = $event"
          @update:include-settings="includeSettings = $event"
          @jump="jumpToSetting"
        />
      </div>

      <!-- 计划模式 -->
      <PlanView v-if="mode === 'plan'" :project-name="projectName" />

      <!-- 审核模式 -->
      <ReviewView v-if="mode === 'review'" :project-id="projectId" />

      <!-- AI 选区工具栏（选择时） -->
      <div
        v-if="aiToolbar.visible && !aiToolbar.result"
        class="ai-toolbar"
        :style="{ left: aiToolbar.x + 'px', top: aiToolbar.y + 'px' }"
      >
        <button @mousedown.prevent @click="runAIAction('continue')">续写</button>
        <button @mousedown.prevent @click="runAIAction('rewrite')">改写</button>
        <button @mousedown.prevent @click="runAIAction('polish')">润色</button>
        <button @mousedown.prevent @click="runAIAction('expand')">扩写</button>
      </div>

      <!-- AI 结果浮动工具栏（插入正文后，等待采用/丢弃/重新生成） -->
      <div
        v-if="aiInsert.active"
        class="ai-insert-toolbar"
        :style="{ left: aiInsert.x + 'px', top: aiInsert.y + 'px' }"
      >
        <span class="ai-insert-label">AI 已生成</span>
        <button @click="regenerateAIResult" :disabled="aiLoading">↻ 重新生成</button>
        <button @click="discardAIResult">丢弃</button>
        <button class="primary" @click="acceptAIResult">采用</button>
      </div>

      <!-- 编辑器右键菜单 -->
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      >
        <div @click="contextAction('highlight')">高亮</div>
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
        <div @click="newDocUnder('chapter')">新建章节</div>
        <div @click="newDocUnder('volume')">新建文件夹</div>
        <div class="divider"></div>
        <div @click="renameDoc">重命名</div>
        <div @click="deleteDoc" class="danger">删除</div>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="status-bar">
      <span class="status-item">{{ wordCount }} 字</span>
      <span class="status-item">{{ saveStatus }}</span>
    </div>

    <!-- 新建文档弹窗 -->
    <CreateDocModal
      v-model="showCreateDocModal"
      :doc-type="createDocType"
      @create="onCreateDocConfirm"
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
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Mark } from '@tiptap/core'
import { documentApi, branchApi, aiApi, projectApi } from '../core/api'
import SearchPanel from '../components/SearchPanel.vue'
import PlanView from '../components/PlanView.vue'
import ReviewView from '../components/ReviewView.vue'
import EditorHeader from '../components/editor/EditorHeader.vue'
import EditorSidebar from '../components/editor/EditorSidebar.vue'
import AIContextPanel from '../components/editor/AIContextPanel.vue'
import CreateDocModal from '../components/editor/CreateDocModal.vue'
import SaveVersionModal from '../components/editor/SaveVersionModal.vue'
import VersionHistoryModal from '../components/editor/VersionHistoryModal.vue'
import { useEditorSearch, createSearchHighlightPlugin } from '../composables/useEditorSearch'

// AI 高亮 Mark — 标记 AI 生成文本（绿色背景）
const AIMark = Mark.create({
  name: 'aiHighlight',
  addAttributes() {
    return {
      action: { default: '' },
    }
  },
  parseHTML() {
    return [{ tag: 'span[data-ai-mark]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['span', { ...HTMLAttributes, 'data-ai-mark': '', class: 'ai-highlight-mark' }, 0]
  },
})

// AI 删除线 Mark — 标记原文（灰色删除线）
const AIStrikethrough = Mark.create({
  name: 'aiStrikethrough',
  parseHTML() {
    return [{ tag: 'span[data-ai-strike]' }]
  },
  renderHTML() {
    return ['span', { 'data-ai-strike': '', class: 'ai-strike-mark' }, 0]
  },
})

const route = useRoute()
const router = useRouter()

const projectId = Number(route.params.projectId)

const editor = shallowRef<any>(null)
const documents = ref<any[]>([])
const branches = ref<any[]>([])
const activeBranchId = ref<number | null>(null)
const versions = ref<any[]>([])
const selectedVersion = ref<any | null>(null)
const compareActive = ref(false)
const compareOldContent = ref('')
const compareCurrentContent = ref('')
const compareSyncScroll = ref(true)
const compareRows = computed(() => buildCompareRows(compareOldContent.value, compareCurrentContent.value))
const compareOldRef = ref<HTMLElement | null>(null)
const compareCurrentRef = ref<HTMLTextAreaElement | null>(null)
let syncingCompareScroll = false
const versionLabel = ref('第一版v1.0')
const loadingVersions = ref(false)
const savingVersion = ref(false)
const restoringVersion = ref(false)
const activeDocId = ref<number | null>(null)
const currentDocName = ref('')
const projectName = ref('')
const loadingDocs = ref(false)
const wordCount = ref(0)
const saveStatus = ref('')
const aiLoading = ref(false)

type AIAction = 'continue' | 'rewrite' | 'polish' | 'expand'

const aiToolbar = ref({ visible: false, x: 0, y: 0, text: '', start: 0, end: 0, action: '' as AIAction | '', result: '' })
// AI 结果已插入正文后的浮动工具栏状态
const aiInsert = ref({
  active: false,   // 是否有 AI 文本在等待用户决定
  x: 0, y: 0,      // 浮动工具栏位置
  from: 0, to: 0,  // AI 文本在文档中的位置
  action: '' as AIAction | '',
  originalText: '',  // 选中的原文（丢弃时恢复）
  originalFrom: 0, originalTo: 0, // 原文的位置
})

const contextMenu = ref({ visible: false, x: 0, y: 0 })
const docContextMenu = ref({ visible: false, x: 0, y: 0, doc: null as any })
const showCreateDocModal = ref(false)
const showSaveVersionModal = ref(false)
const showVersionModal = ref(false)
const newDocName = ref('')
const createDocType = ref<'volume' | 'chapter'>('chapter')
const createDocParent = ref<number | null>(null)

let saveTimer: any = null
let selectionHandler: (() => void) | null = null
const isSettingContent = ref(false)

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

// 左侧栏 Tab：章节 / 设定
const sideTab = ref<'chapter' | 'setting'>('chapter')
const CHAPTER_TYPES = ['volume', 'chapter']
const SETTING_TYPES = ['outline', 'characters', 'worldbuilding']

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
const includeSettings = ref(true)  // 是否让 AI 参考项目设定
const contextPanelOpen = ref(true) // 右侧参考面板展开/收起

async function jumpToSetting(docId: number) {
  const doc = documents.value.find(d => d.id === docId)
  if (doc) {
    sideTab.value = 'setting'
    await openDoc(doc, { keepSearch: true })
  }
}

function switchMode(next: EditorMode) {
  if (mode.value === next) return
  mode.value = next
  // 模式联动侧栏：计划 → 设定；写作/审核 → 章节
  if (next === 'plan') sideTab.value = 'setting'
  else sideTab.value = 'chapter'
  // 关闭浮层，避免残留
  aiToolbar.value.visible = false
  aiInsert.value.active = false
  contextMenu.value.visible = false
  if (searchVisible.value) closeSearch()
}

const filteredRootNodes = computed(() => {
  const types = sideTab.value === 'chapter' ? CHAPTER_TYPES : SETTING_TYPES
  return rootNodes.value.filter(n => types.includes(n.type))
})

// 沿 parent_id 向上找到根节点的 type
function findRootType(doc: any): string | null {
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
  return cur?.type || null
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
      AIMark,
      AIStrikethrough,
    ],
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
    // AI 结果预览期间不响应选区变化，防止清空预览
    if (aiToolbar.value.result) return
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
  const { from, to } = editor.value.state.selection
  const text = editor.value.state.doc.textBetween(from, to, '\n')
  return { text, start: from, end: to }
}

function showAIToolbar(sel: any) {
  const nativeSel = window.getSelection()
  if (!nativeSel || nativeSel.rangeCount === 0) return
  const range = nativeSel.getRangeAt(0)
  const rect = range.getBoundingClientRect()
  aiToolbar.value = {
    visible: true,
    x: Math.max(8, rect.left + rect.width / 2 - 80),
    y: Math.max(8, rect.top - 50),
    text: sel.text,
    start: sel.start,
    end: sel.end,
    action: '',
    result: ''
  }
}

function buildCompareRows(oldContent: string, currentContent: string) {
  const oldLines = oldContent.split('\n')
  const currentLines = currentContent.split('\n')
  const max = Math.max(oldLines.length, currentLines.length)
  return Array.from({ length: max }, (_, index) => {
    const oldText = oldLines[index] ?? ''
    const currentText = currentLines[index] ?? ''
    let type = 'same'
    if (oldText !== currentText) {
      type = oldText && currentText ? 'changed' : oldText ? 'removed' : 'added'
    }
    return { oldText: oldText || (type === 'added' ? '＋ 当前版本新增此行' : ''), currentText, type }
  })
}

function editorContentFromText(content: string) {
  return content
    ? { type: 'doc', content: content.split('\n').map((line: string) => ({
        type: 'paragraph',
        content: line ? [{ type: 'text', text: line }] : []
      })) }
    : { type: 'doc', content: [{ type: 'paragraph' }] }
}

function getEditorText(ed = editor.value) {
  if (!ed) return ''
  const blocks: string[] = []
  ed.state.doc.forEach((node: any) => {
    blocks.push(node.textContent || '')
  })
  return blocks.join('\n')
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
  const rootType = findRootType(doc)
  if (rootType && SETTING_TYPES.includes(rootType)) sideTab.value = 'setting'
  else if (rootType && CHAPTER_TYPES.includes(rootType)) sideTab.value = 'chapter'
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
      const tiptapDoc = editorContentFromText(content)
      editor.value.commands.setContent(tiptapDoc, { emitUpdate: false })
      wordCount.value = content.length
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
      content: getEditorText()
    })
    saveStatus.value = '已保存'
  } catch (e) {
    saveStatus.value = '保存失败'
  }
}

function openCreateDocModal(parentId: number | null = null, type: 'volume' | 'chapter' = 'chapter') {
  createDocParent.value = parentId
  createDocType.value = type
  newDocName.value = ''
  showCreateDocModal.value = true
}

// 拆分组件 CreateDocModal 触发（emit create）
async function onCreateDocConfirm(name: string) {
  newDocName.value = name
  await createDoc()
}

async function createDoc() {
  if (!newDocName.value.trim()) return
  try {
    // 计算 sort_order
    const siblings = documents.value.filter(d =>
      (createDocParent.value === null ? !d.parent_id : d.parent_id === createDocParent.value)
    )
    const maxSort = siblings.reduce((m, d) => Math.max(m, d.sort_order || 0), 0)

    const res = await documentApi.create({
      project_id: projectId,
      branch_id: activeBranchId.value || undefined,
      title: newDocName.value.trim(),
      type: createDocType.value,
      parent_id: createDocParent.value || undefined,
      sort_order: maxSort + 1,
    })
    showCreateDocModal.value = false
    newDocName.value = ''
    await loadDocs()
    const newDoc = res.data?.data || res.data
    if (newDoc && createDocType.value === 'chapter') {
      openDoc(newDoc)
    }
  } catch (e: any) {
    alert(e.response?.data?.error || '创建失败')
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

function newDocUnder(type: 'volume' | 'chapter') {
  const doc = docContextMenu.value.doc
  if (!doc) return
  // 如果右键的节点是文件夹，就在它下面创建；否则在它的同级创建
  const isFolder = ['volume', 'outline', 'characters', 'worldbuilding'].includes(doc.type)
  const parentId = isFolder ? doc.id : (doc.parent_id || null)
  openCreateDocModal(parentId, type)
  closeDocContextMenu()
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
    alert(e.response?.data?.error || '重命名失败')
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
    alert(e.response?.data?.error || '删除失败')
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
    alert(e.response?.data?.error || '移动失败')
  }
}

async function runAIAction(action: AIAction) {
  if (!editor.value || !aiToolbar.value.text || aiLoading.value) return
  // 如果上一次 AI 插入还在等待决定，先忽略新请求
  if (aiInsert.value.active) return
  aiLoading.value = true
  const oldStatus = saveStatus.value
  saveStatus.value = 'AI 生成中...'
  const { start, end, text: selectedText } = aiToolbar.value
  // 隐藏选区工具栏
  aiToolbar.value.visible = false
  try {
    const res = await aiApi.generate({
      project_id: String(projectId),
      document_id: activeDocId.value ? String(activeDocId.value) : undefined,
      mode: action,
      content: buildAIPrompt(action, selectedText),
      stream: false,
      include_settings: includeSettings.value,
    })
    const data = res.data?.data || res.data
    const result = cleanAIResult(data?.content || '')
    if (!result) throw new Error('AI 返回为空')

    // 保存本次 AI 参考的设定（用于右侧面板展示）
    if (Array.isArray(data?.used_settings)) {
      usedSettings.value = data.used_settings.map((s: any) => ({
        id: s.id, title: s.title, type: s.type,
        used: !!s.used, tokens: s.tokens || 0,
      }))
    }

    // 直接插入编辑器正文
    const isContinue = action === 'continue'

    // 记住原文信息
    aiInsert.value = {
      active: true,
      x: 0, y: 0,
      from: 0, to: 0,
      action,
      originalText: selectedText,
      originalFrom: start,
      originalTo: end,
    }

    if (isContinue) {
      // 续写：选区保留，在后面追加 AI 文本
      const insertText = `\n${result}`
      editor.value.chain()
        .focus()
        .setTextSelection({ from: end, to: end })
        .insertContent(insertText)
        .run()
      aiInsert.value.from = end + 1
      aiInsert.value.to = end + insertText.length
      // 给 AI 文本加绿色高亮 mark，并确保不继承原文删除线
      removeTextMark('aiStrikethrough', aiInsert.value.from, aiInsert.value.to)
      applyTextMark('aiHighlight', aiInsert.value.from, aiInsert.value.to, { action })
      editor.value.commands.setTextSelection(aiInsert.value.to)
    } else {
      // 改写/润色/扩写：
      // 1) 原文保留，对选区内所有 text node 加删除线 mark
      // 2) 在原文后面追加 AI 文本 + 绿色高亮 mark
      const originalRange = expandRangeToTextBoundaries(start, end)
      aiInsert.value.originalFrom = originalRange.from
      aiInsert.value.originalTo = originalRange.to
      applyTextMark('aiStrikethrough', originalRange.from, originalRange.to)

      // 在原文后插入换行 + AI 结果
      const insertText = `\n${result}`
      editor.value.chain()
        .setTextSelection({ from: originalRange.to, to: originalRange.to })
        .insertContent(insertText)
        .run()

      aiInsert.value.from = originalRange.to + 1
      aiInsert.value.to = originalRange.to + insertText.length

      // AI 文本只加绿色高亮，强制清掉可能继承的原文删除线
      removeTextMark('aiStrikethrough', aiInsert.value.from, aiInsert.value.to)
      applyTextMark('aiHighlight', aiInsert.value.from, aiInsert.value.to, { action })
      editor.value.commands.setTextSelection(aiInsert.value.to)
    }

    // 计算浮动工具栏位置
    nextTick(() => {
      const coords = editor.value.view.coordsAtPos(aiInsert.value.to)
      aiInsert.value.x = coords.left + 10
      aiInsert.value.y = coords.bottom + 8
    })

    saveStatus.value = 'AI 已生成，等待采用'
  } catch (e: any) {
    console.error('AI 生成失败', e)
    saveStatus.value = oldStatus || 'AI 生成失败'
    alert(e.response?.data?.error || e.message || 'AI 生成失败')
    aiInsert.value.active = false
  } finally {
    aiLoading.value = false
  }
}

function buildAIPrompt(action: AIAction, selectedText: string) {
  const instructions: Record<AIAction, string> = {
    continue: '你是小说写作助手。请直接续写下面的小说文本，只输出续写的正文内容。不要解释、不要加标题、不要输出多个版本、不要分析。直接开始写小说正文。',
    rewrite: '你是小说编辑。请直接改写下面的小说文本，保留核心情节，提升表达。只输出改写后的正文，不要解释、不要分析、不要给多个版本。直接输出改写后的最终文本。',
    polish: '你是文字编辑。请直接润色下面的小说文本，保持原意，修正语病，提升文字质感。只输出润色后的正文，不要解释、不要分析、不要给多个版本。直接输出最终文本。',
    expand: '你是小说作家。请直接扩写下面的小说文本，增加细节、氛围、动作和心理描写。只输出扩写后的正文，不要解释、不要分析、不要给多个版本。直接输出最终文本。',
  }
  return `${instructions[action]}\n\n${selectedText}`
}

function cleanAIResult(text: string) {
  return String(text || '')
    .replace(/^```[\s\S]*?\n/, '')
    .replace(/```$/g, '')
    .trim()
}

function expandRangeToTextBoundaries(from: number, to: number) {
  if (!editor.value || from >= to) return { from, to }
  let expandedFrom = from
  let expandedTo = to
  editor.value.state.doc.nodesBetween(from, to, (node: any, pos: number) => {
    if (!node.isText) return true
    const nodeFrom = pos
    const nodeTo = pos + node.nodeSize
    // 如果选区与这个 text node 有交集，就用 text node 边界兜底。
    // 这样能避免 ProseMirror 光标位置落在句首字符后面时，首字漏掉删除线。
    if (nodeTo > from && nodeFrom < to) {
      expandedFrom = Math.min(expandedFrom, nodeFrom)
      expandedTo = Math.max(expandedTo, nodeTo)
    }
    return true
  })
  return { from: expandedFrom, to: expandedTo }
}

function applyTextMark(markName: string, from: number, to: number, attrs: Record<string, any> = {}) {
  if (!editor.value || from >= to) return
  const { state, view } = editor.value
  const markType = state.schema.marks[markName]
  if (!markType) return
  let tr = state.tr
  state.doc.nodesBetween(from, to, (node: any, pos: number) => {
    if (!node.isText) return true
    const start = Math.max(from, pos)
    const end = Math.min(to, pos + node.nodeSize)
    if (start < end) {
      tr = tr.addMark(start, end, markType.create(attrs))
    }
    return true
  })
  if (tr.docChanged) view.dispatch(tr)
}

function removeTextMark(markName: string, from: number, to: number) {
  if (!editor.value || from >= to) return
  const { state, view } = editor.value
  const markType = state.schema.marks[markName]
  if (!markType) return
  let tr = state.tr
  state.doc.nodesBetween(from, to, (node: any, pos: number) => {
    if (!node.isText) return true
    const start = Math.max(from, pos)
    const end = Math.min(to, pos + node.nodeSize)
    if (start < end) {
      tr = tr.removeMark(start, end, markType)
    }
    return true
  })
  if (tr.docChanged) view.dispatch(tr)
}

function acceptAIResult() {
  if (!editor.value || !aiInsert.value.active) return
  const { from, to, originalFrom, action } = aiInsert.value

  if (action === 'continue') {
    // 续写：保留 AI 文本，只去掉绿色高亮
    removeTextMark('aiHighlight', from, to)
    editor.value.commands.setTextSelection(to)
  } else {
    // 改写/润色/扩写：采用后才真正删除原文（含原文后的换行），保留 AI 文本为普通正文
    const aiText = editor.value.state.doc.textBetween(from, to, '\n')
    editor.value.chain()
      .focus()
      .deleteRange({ from: originalFrom, to })
      .insertContentAt({ from: originalFrom, to: originalFrom }, aiText)
      .setTextSelection(originalFrom + aiText.length)
      .run()
  }

  aiInsert.value.active = false
  saveStatus.value = '编辑中...'
}

function discardAIResult() {
  if (!editor.value || !aiInsert.value.active) return
  const { to, originalFrom, originalTo, action } = aiInsert.value

  // 删除 AI 文本；同时删除 AI 前面那一个换行，所以从 originalTo 开始删到 AI 末尾
  editor.value.chain()
    .focus()
    .deleteRange({ from: originalTo, to })
    .run()

  // 非续写模式：原文还在，只需要去掉删除线 mark
  if (action !== 'continue') {
    removeTextMark('aiStrikethrough', originalFrom, originalTo)
    editor.value.commands.setTextSelection(originalTo)
  }

  aiInsert.value.active = false
  saveStatus.value = '编辑中...'
}

async function regenerateAIResult() {
  if (!editor.value || !aiInsert.value.active) return
  const { action, originalText } = aiInsert.value
  // 先丢弃当前结果
  discardAIResult()
  // 重新设置 aiToolbar 状态再调用
  aiToolbar.value = {
    visible: false,
    x: 0, y: 0,
    text: originalText,
    start: aiInsert.value.originalFrom,
    end: aiInsert.value.originalTo,
    action,
    result: '',
  }
  aiInsert.value.active = false // discardAIResult 已设，再保险
  await runAIAction(action as AIAction)
}

function openContextMenu(e: MouseEvent) {
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY }
  document.addEventListener('click', closeContextMenu, { once: true })
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function contextAction(action: string) {
  if (!editor.value) return
  const sel = getSelection()
  if (!sel.text && action !== 'paste') return
  switch (action) {
    case 'highlight':
      editor.value.commands.insertContentAt(
        { from: sel.start, to: sel.end },
        `==${sel.text}==`
      )
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

function openSaveVersionModal() {
  if (!activeDocId.value || !editor.value) return
  versionLabel.value = versionLabel.value || '第一版v1.0'
  showSaveVersionModal.value = true
}

// 拆分组件 SaveVersionModal 触发（emit save）
async function onSaveVersionConfirm(label: string) {
  versionLabel.value = label
  await saveVersion(label)
}

async function saveVersion(label: string) {
  if (!activeDocId.value || !editor.value || savingVersion.value) return
  versionLabel.value = label.trim()
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
  savingVersion.value = true
  try {
    saveStatus.value = '正在保存版本...'
    await documentApi.update(activeDocId.value, {
      content: getEditorText(),
      create_version: true,
      version_label: versionLabel.value,
    })
    saveStatus.value = '版本已保存'
    showSaveVersionModal.value = false
    await loadDocs()
    if (showVersionModal.value) await loadVersions(true)
  } catch (e: any) {
    console.error('保存版本失败', e)
    saveStatus.value = '保存版本失败'
    alert(e.response?.data?.error || '保存版本失败')
  } finally {
    savingVersion.value = false
  }
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
}

// 搜索的所有函数抽到 composables/useEditorSearch.ts

async function openVersionModal() {
  if (!activeDocId.value) return
  showVersionModal.value = true
  await loadVersions(true)
}

async function loadVersions(selectFirst = false) {
  if (!activeDocId.value) return
  loadingVersions.value = true
  selectedVersion.value = null
  try {
    const res = await documentApi.listVersions(activeDocId.value)
    versions.value = res.data?.data || res.data || []
    if (selectFirst && versions.value.length > 0) {
      await selectVersion(versions.value[0])
    }
  } catch (e: any) {
    console.error('加载版本历史失败', e)
    versions.value = []
    alert(e.response?.data?.error || '加载版本历史失败')
  } finally {
    loadingVersions.value = false
  }
}

async function selectVersion(v: any) {
  try {
    const res = await documentApi.getVersion(v.id)
    selectedVersion.value = res.data?.data || res.data
  } catch (e: any) {
    alert(e.response?.data?.error || '加载版本失败')
  }
}

async function startCompareWithCurrent() {
  if (!selectedVersion.value || !editor.value) return
  compareOldContent.value = selectedVersion.value.content_snapshot || ''
  compareCurrentContent.value = getEditorText()
  compareSyncScroll.value = true
  compareActive.value = true
  showVersionModal.value = false
  aiToolbar.value.visible = false
}

function closeCompareView() {
  compareActive.value = false
  compareOldContent.value = ''
  compareCurrentContent.value = ''
}

function onCompareCurrentInput() {
  wordCount.value = compareCurrentContent.value.length
  saveStatus.value = '编辑中...'
}

function syncCompareScroll(source: 'old' | 'current') {
  if (!compareSyncScroll.value || syncingCompareScroll) return
  const oldEl = compareOldRef.value
  const currentEl = compareCurrentRef.value
  if (!oldEl || !currentEl) return
  const from = source === 'old' ? oldEl : currentEl
  const to = source === 'old' ? currentEl : oldEl
  syncingCompareScroll = true
  to.scrollTop = from.scrollTop
  to.scrollLeft = from.scrollLeft
  requestAnimationFrame(() => {
    syncingCompareScroll = false
  })
}

async function applyCompareCurrent() {
  if (!activeDocId.value || !editor.value) return
  isSettingContent.value = true
  const content = compareCurrentContent.value
  editor.value.commands.setContent(editorContentFromText(content), { emitUpdate: false })
  wordCount.value = content.length
  setTimeout(() => { isSettingContent.value = false }, 100)
  compareActive.value = false
  await autoSave()
}

async function restoreSelectedVersion() {
  if (!selectedVersion.value || !editor.value) return
  if (!confirm('确定恢复此版本？当前内容会先保存为一个历史版本。')) return
  restoringVersion.value = true
  try {
    const res = await documentApi.restoreVersion(selectedVersion.value.id)
    const data = res.data?.data || res.data
    const content = data?.content || ''
    compareActive.value = false
    isSettingContent.value = true
    const tiptapDoc = editorContentFromText(content)
    editor.value.commands.setContent(tiptapDoc, { emitUpdate: false })
    wordCount.value = content.length
    saveStatus.value = '已恢复版本'
    setTimeout(() => { isSettingContent.value = false }, 100)
    await loadDocs()
    await loadVersions(true)
  } catch (e: any) {
    alert(e.response?.data?.error || '恢复版本失败')
  } finally {
    restoringVersion.value = false
  }
}

function goBack() {
  router.push('/bookshelf')
}
</script>

<style scoped>
.editor-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  max-height: 100vh;
  background: var(--bg-primary);
  overflow: hidden;
}

.editor-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 6px 24px;
  background: var(--glass-bg-solid);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
  gap: 16px;
  z-index: 10;
}

.back-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-secondary);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.back-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.doc-name {
  flex: 1;
  min-width: 0;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: baseline;
  gap: 8px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.doc-name .proj-name {
  color: var(--text-primary);
  font-weight: 600;
  flex-shrink: 0;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
}
.doc-name .doc-sep {
  color: var(--text-tertiary, #aeaeb2);
  font-weight: 400;
  flex-shrink: 0;
}
.doc-name .chapter-name {
  color: var(--text-secondary, #86868b);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* 顶部三模式切换器（iOS 分段控件） */
.mode-switcher {
  display: inline-flex;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 10px;
  padding: 3px;
  gap: 2px;
  margin: 0 12px;
}
.mode-tab {
  border: none;
  background: transparent;
  padding: 6px 16px;
  font-size: 13px;
  color: var(--text-secondary);
  border-radius: 7px;
  cursor: pointer;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s ease;
}
.mode-tab .dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: currentColor;
  opacity: 0.7;
}
.mode-tab:hover:not(.active) { color: var(--text-primary); }
.mode-tab.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  font-weight: 600;
}
.mode-tab.plan.active { color: #af52de; }
.mode-tab.write.active { color: #007aff; }
.mode-tab.review.active { color: #ff9500; }

.btn {
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
  transition: all var(--transition-fast);
}

.btn:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn.small {
  padding: 4px 10px;
  font-size: 12px;
}

.btn.primary {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}

.editor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* 左侧文档树 */
.doc-tree {
  width: 260px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary);
}

.tree-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-subtle);
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
}

/* 左侧栏 章节/设定 Tab */
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

.tree-empty-hint {
  display: block;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-tertiary);
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
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-secondary);
}

.branch-switcher select {
  flex: 1;
  min-width: 0;
  height: 28px;
  border: 1px solid var(--border);
  border-radius: var(--radius-xs);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 12px;
  padding: 0 8px;
  outline: none;
}

.add-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: var(--accent);
  color: #fff;
  border-radius: var(--radius-xs);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  transition: background var(--transition-fast);
}

.add-btn:hover {
  background: var(--accent-hover);
}

.tree-list {
  flex: 1;
  overflow: auto;
  padding: 8px 0;
}

.tree-loading, .tree-empty {
  text-align: center;
  color: var(--text-secondary);
  padding: 24px 16px;
  font-size: 13px;
  line-height: 2;
}

/* 编辑器 */
.editor-main {
  flex: 1;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.editor-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary);
  font-size: 15px;
}

.compare-view {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
}

.compare-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-secondary);
}

.compare-topbar strong {
  color: var(--text-primary);
  font-size: 14px;
  margin-right: 10px;
}

.compare-topbar span {
  color: var(--text-secondary);
  font-size: 12px;
}

.compare-actions,
.preview-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sync-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}

.sync-toggle input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.sync-toggle i {
  position: relative;
  display: inline-block;
  width: 38px;
  height: 22px;
  border-radius: 999px;
  background: #d1d1d6;
  transition: background var(--transition-fast);
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.04);
}

.sync-toggle i::before {
  content: '';
  position: absolute;
  width: 18px;
  height: 18px;
  left: 2px;
  top: 2px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.22);
  transition: transform var(--transition-fast);
}

.sync-toggle input:checked + i {
  background: var(--accent, #007aff);
}

.sync-toggle input:checked + i::before {
  transform: translateX(16px);
}

.compare-panes {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1px;
  background: var(--border-subtle);
}

.compare-pane {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
}

.pane-title {
  flex-shrink: 0;
  padding: 10px 16px;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-subtle);
}

.diff-lines,
.compare-pane textarea {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 18px 20px;
  border: 0;
  outline: none;
  resize: none;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-serif);
  font-size: 16px;
  line-height: 1.9;
  color: var(--text-primary);
  background: var(--bg-primary);
}

.diff-line {
  min-height: 1.9em;
  padding: 0 8px;
  margin: 0 -8px 2px;
  border-radius: 6px;
}

.diff-line.changed {
  background: rgba(255, 149, 0, 0.14);
  color: #8a4b00;
}

.diff-line.removed {
  background: rgba(255, 59, 48, 0.12);
  color: #b42318;
  text-decoration: line-through;
}

.diff-line.added {
  background: rgba(52, 199, 89, 0.12);
  color: #1f7a3a;
  font-family: var(--font-sans, -apple-system, 'SF Pro', sans-serif);
  font-size: 13px;
}

.compare-pane.old .diff-lines {
  color: var(--text-secondary);
  background: #fafafa;
}

.compare-pane.current textarea {
  background: #fff;
}

.editor-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tiptap-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 60px 24px 60px 40px;
}

.tiptap-container > :deep(div) {
  max-width: 880px;
  margin: 0;
}

.tiptap-container > :deep(div .ProseMirror) {
  max-width: none;
}

:deep(.ProseMirror) {
  outline: none;
  min-height: 500px;
  font-family: var(--font-serif);
  font-size: 17px;
  line-height: 1.9;
  color: var(--text-primary);
  overflow: visible !important;
  height: auto !important;
  counter-reset: para;
}

:deep(.ProseMirror) > p,
:deep(.ProseMirror) > h1,
:deep(.ProseMirror) > h2,
:deep(.ProseMirror) > h3,
:deep(.ProseMirror) > blockquote,
:deep(.ProseMirror) > ul,
:deep(.ProseMirror) > ol,
:deep(.ProseMirror) > pre {
  position: relative;
  counter-increment: para;
}

:deep(.ProseMirror) > p::before,
:deep(.ProseMirror) > h1::before,
:deep(.ProseMirror) > h2::before,
:deep(.ProseMirror) > h3::before,
:deep(.ProseMirror) > blockquote::before,
:deep(.ProseMirror) > ul::before,
:deep(.ProseMirror) > ol::before,
:deep(.ProseMirror) > pre::before {
  content: counter(para);
  position: absolute;
  left: -2.6em;
  top: 1.32em;
  width: 1.6em;
  text-align: center;
  font-family: var(--font-sans, -apple-system, 'SF Pro', sans-serif);
  font-size: 12px;
  line-height: 1;
  color: var(--text-tertiary, #c7c7cc);
  font-variant-numeric: tabular-nums;
  user-select: none;
  pointer-events: none;
  font-weight: 400;
}

/* 占位段落（空文档）不显示行号，避免空 1 突兀 */
:deep(.ProseMirror) > p.is-editor-empty::before {
  content: '';
}

:deep(.ProseMirror p) {
  margin-bottom: 0.8em;
}

:deep(.ProseMirror h1) {
  font-size: 28px;
  font-weight: 700;
  margin: 1em 0 0.6em;
}

:deep(.ProseMirror h2) {
  font-size: 22px;
  font-weight: 600;
  margin: 1em 0 0.5em;
}

:deep(.ProseMirror p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  color: var(--text-tertiary);
  pointer-events: none;
  float: left;
  height: 0;
}

/* AI 工具栏 */
.ai-toolbar {
  position: absolute;
  background: rgba(29, 29, 31, 0.95);
  backdrop-filter: blur(12px);
  color: #fff;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  gap: 2px;
  z-index: 200;
  box-shadow: var(--shadow-lg);
}

.ai-toolbar button {
  padding: 6px 14px;
  background: none;
  border: none;
  color: #fff;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  border-radius: var(--radius-xs);
  transition: background var(--transition-fast);
}

.ai-toolbar button:hover {
  background: rgba(255, 255, 255, 0.15);
}

/* AI 高亮 Mark — 编辑器正文内 AI 生成文本的样式 */
:deep(.ai-highlight-mark) {
  display: inline;
  background: rgba(52, 199, 89, 0.14);
  border-radius: 4px;
  padding: 1px 3px;
  box-shadow: 0 0 0 1px rgba(52, 199, 89, 0.12);
}

/* AI 删除线 Mark — 原文等待替换时的样式 */
:deep(.ai-strike-mark) {
  color: var(--text-tertiary, #aeaeb2);
  text-decoration: line-through;
  text-decoration-thickness: 1px;
  text-decoration-color: rgba(142, 142, 147, 0.9);
  background: rgba(142, 142, 147, 0.08);
  border-radius: 4px;
  padding: 1px 3px;
}

/* AI 结果浮动工具栏（插入正文后） */
.ai-insert-toolbar {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-secondary, #fff);
  border: 1px solid var(--border, #e0e0e0);
  border-radius: var(--radius-sm, 8px);
  box-shadow: 0 4px 16px rgba(0,0,0,0.1);
  padding: 5px 8px;
  z-index: 400;
  font-size: 13px;
}

.ai-insert-label {
  color: var(--text-secondary, #86868b);
  font-size: 12px;
  margin-right: 2px;
}

.ai-insert-toolbar button {
  border: 1px solid var(--border, #e0e0e0);
  background: var(--bg-secondary, #fff);
  color: var(--text-primary, #1d1d1f);
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
}

.ai-insert-toolbar button:hover:not(:disabled) {
  opacity: 0.85;
}

.ai-insert-toolbar button:disabled {
  opacity: 0.4;
  cursor: wait;
}

.ai-insert-toolbar button.primary {
  background: var(--accent, #007aff);
  color: #fff;
  border-color: var(--accent, #007aff);
}

/* 右键菜单 */
.context-menu {
  position: absolute;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 4px 0;
  box-shadow: var(--shadow-lg);
  z-index: 300;
  min-width: 140px;
}

.context-menu div {
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
  transition: background var(--transition-fast);
}

.context-menu div:hover {
  background: var(--accent-light);
  color: var(--accent);
}

.context-menu div.danger {
  color: #ff3b30;
}

.context-menu div.danger:hover {
  background: rgba(255, 59, 48, 0.1);
  color: #ff3b30;
}

.context-menu .divider {
  height: 1px;
  background: var(--border-subtle);
  margin: 4px 0;
  padding: 0;
}

.context-menu .divider:hover {
  background: var(--border-subtle);
}

/* 状态栏 */
.status-bar {
  flex-shrink: 0;
  height: var(--status-bar-height);
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 24px;
  background: var(--bg-tertiary);
  border-top: 1px solid var(--border-subtle);
  font-size: 12px;
  color: var(--text-secondary);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 500;
}

.modal {
  background: var(--bg-secondary);
  padding: 24px;
  border-radius: var(--radius-lg);
  width: 400px;
  max-height: 80vh;
  overflow: auto;
  box-shadow: var(--shadow-xl);
}

.modal h3 {
  font-family: var(--font-serif);
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 16px;
  color: var(--text-primary);
}

.input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-primary);
  font-size: 14px;
  margin-bottom: 16px;
  outline: none;
  transition: all var(--transition-fast);
}

.input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-light);
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.modal-tip {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
  margin: -4px 0 14px;
}

.save-version-modal {
  width: 420px;
}

.version-modal {
  width: min(960px, 92vw);
  height: min(680px, 86vh);
  display: flex;
  flex-direction: column;
  padding: 0;
}

.version-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 20px 22px 14px;
  border-bottom: 1px solid var(--border-subtle);
}

.version-header h3 {
  margin-bottom: 4px;
}

.version-header p {
  color: var(--text-secondary);
  font-size: 12px;
}

.icon-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 22px;
  line-height: 1;
  padding: 2px 6px;
  border-radius: var(--radius-xs);
}

.icon-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.version-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px 1fr;
}

.version-list {
  border-right: 1px solid var(--border-subtle);
  overflow: auto;
  padding: 8px;
}

.version-item {
  padding: 12px;
  border: 1px solid transparent;
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast), border-color var(--transition-fast);
}

.version-item:hover,
.version-item.active {
  background: var(--accent-subtle);
  border-color: var(--accent-light);
}

.version-time {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
}

.version-item .note {
  color: var(--text-secondary);
  font-size: 12px;
  margin-top: 4px;
}

.version-preview {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.preview-toolbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: 12px;
}

.version-preview pre {
  flex: 1;
  margin: 0;
  padding: 18px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-serif);
  font-size: 15px;
  line-height: 1.8;
  color: var(--text-primary);
  background: var(--bg-primary);
}

.empty-tip {
  text-align: center;
  color: var(--text-secondary);
  padding: 24px;
}

/* 搜索命中高亮（ProseMirror Decoration） */
:deep(.ProseMirror) .search-hit {
  background: rgba(255, 214, 10, 0.35);
  border-radius: 2px;
  transition: background 0.15s ease;
}
:deep(.ProseMirror) .search-hit-active {
  background: rgba(255, 149, 0, 0.55);
  box-shadow: 0 0 0 1px rgba(255, 149, 0, 0.9);
}

/* AI 参考面板（写作模式右侧浮层） */
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