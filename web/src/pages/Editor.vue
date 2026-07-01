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
import { ref, shallowRef, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { documentApi, branchApi, projectApi } from '../core/api'
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
import { useAIToolbar, AIMark, AIStrikethrough } from '../composables/useAIToolbar'
import { useVersionManager } from '../composables/useVersionManager'

// AI Marks (AIMark/AIStrikethrough) 抽到 composables/useAIToolbar.ts

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

// AI 状态（useAIToolbar 调用挪到 usedSettings 定义之后）

const contextMenu = ref({ visible: false, x: 0, y: 0 })
const docContextMenu = ref({ visible: false, x: 0, y: 0, doc: null as any })
const showCreateDocModal = ref(false)
const newDocName = ref('')
const createDocType = ref<'volume' | 'chapter'>('chapter')
const createDocParent = ref<number | null>(null)

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

// AI 工具栏 & 交互 —— composable
const {
  aiLoading,
  aiToolbar,
  aiInsert,
  showAIToolbar,
  runAIAction,
  acceptAIResult,
  discardAIResult,
  regenerateAIResult,
} = useAIToolbar({
  editor,
  projectId,
  activeDocId,
  includeSettings,
  usedSettings,
  saveStatus,
})

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

// showAIToolbar 抽到 composables/useAIToolbar.ts

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

// AI 相关函数 (runAIAction/buildAIPrompt/cleanAIResult/expandRangeToTextBoundaries/
//   applyTextMark/removeTextMark/acceptAIResult/discardAIResult/regenerateAIResult)
// 抽到 composables/useAIToolbar.ts

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

function goBack() {
  router.push('/bookshelf')
}
</script>

<style scoped src="./editor-page.css"></style>
