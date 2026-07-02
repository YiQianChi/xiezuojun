import { ref, computed, type Ref, type ShallowRef } from 'vue'
import { documentApi } from '../core/api'
import { toast } from './useToast'

interface UseVersionManagerOptions {
  editor: ShallowRef<any>
  activeDocId: Ref<number | null>
  isSettingContent: Ref<boolean>
  wordCount: Ref<number>
  saveStatus: Ref<string>
  autoSave: () => Promise<void>
  loadDocs: () => Promise<void>
  getEditorText: () => string
  editorContentFromText: (content: string) => any
  /** hide AI toolbar when entering compare view */
  hideAIToolbar?: () => void
  /** cancel pending auto-save timer before version save */
  cancelSaveTimer?: () => void
}

export function useVersionManager(opts: UseVersionManagerOptions) {
  const {
    editor, activeDocId, isSettingContent,
    wordCount, saveStatus,
    autoSave, loadDocs, getEditorText, editorContentFromText,
    hideAIToolbar, cancelSaveTimer,
  } = opts

  // ── 版本状态 ──
  const versions = ref<any[]>([])
  const selectedVersion = ref<any | null>(null)
  const versionLabel = ref('第一版v1.0')
  const loadingVersions = ref(false)
  const savingVersion = ref(false)
  const restoringVersion = ref(false)
  const showVersionModal = ref(false)
  const showSaveVersionModal = ref(false)

  // ── 版本对比状态 ──
  const compareActive = ref(false)
  const compareOldContent = ref('')
  const compareCurrentContent = ref('')
  const compareSyncScroll = ref(true)
  const compareOldRef = ref<HTMLElement | null>(null)
  const compareCurrentRef = ref<HTMLTextAreaElement | null>(null)
  let syncingCompareScroll = false

  const compareRows = computed(() =>
    buildCompareRows(compareOldContent.value, compareCurrentContent.value)
  )

  // ── 保存版本 ──
  function openSaveVersionModal() {
    if (!activeDocId.value || !editor.value) return
    versionLabel.value = versionLabel.value || '第一版v1.0'
    showSaveVersionModal.value = true
  }

  /** Bridge: called by SaveVersionModal @save emit */
  async function onSaveVersionConfirm(label: string) {
    versionLabel.value = label
    await saveVersion(label)
  }

  async function saveVersion(label: string) {
    if (!activeDocId.value || !editor.value || savingVersion.value) return
    versionLabel.value = label.trim()
    cancelSaveTimer?.()
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
      toast.error(e.response?.data?.error || '保存版本失败')
    } finally {
      savingVersion.value = false
    }
  }

  // ── 版本历史 ──
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
      toast.error(e.response?.data?.error || '加载版本历史失败')
    } finally {
      loadingVersions.value = false
    }
  }

  async function selectVersion(v: any) {
    try {
      const res = await documentApi.getVersion(v.id)
      selectedVersion.value = res.data?.data || res.data
    } catch (e: any) {
      toast.error(e.response?.data?.error || '加载版本失败')
    }
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
      toast.error(e.response?.data?.error || '恢复版本失败')
    } finally {
      restoringVersion.value = false
    }
  }

  // ── 对比功能 ──
  async function startCompareWithCurrent() {
    if (!selectedVersion.value || !editor.value) return
    compareOldContent.value = selectedVersion.value.content_snapshot || ''
    compareCurrentContent.value = getEditorText()
    compareSyncScroll.value = true
    compareActive.value = true
    showVersionModal.value = false
    hideAIToolbar?.()
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

  return {
    // 版本状态
    versions,
    selectedVersion,
    versionLabel,
    loadingVersions,
    savingVersion,
    restoringVersion,
    showVersionModal,
    showSaveVersionModal,
    // 版本操作
    openSaveVersionModal,
    onSaveVersionConfirm,
    saveVersion,
    openVersionModal,
    loadVersions,
    selectVersion,
    restoreSelectedVersion,
    // 对比状态
    compareActive,
    compareOldContent,
    compareCurrentContent,
    compareSyncScroll,
    compareRows,
    compareOldRef,
    compareCurrentRef,
    // 对比操作
    startCompareWithCurrent,
    closeCompareView,
    onCompareCurrentInput,
    syncCompareScroll,
    applyCompareCurrent,
  }
}

// ── 工具函数 ──

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
