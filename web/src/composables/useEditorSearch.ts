/**
 * useEditorSearch — 编辑器搜索 composable
 *
 * 从 Editor.vue 抽出的搜索逻辑：
 * - 本章搜索（前端 ProseMirror Decoration Plugin，单字符即触发）
 * - 全项目搜索（后端接口 /api/projects/:id/search，≥2 中文/3 英文字符触发，debounce 250ms）
 * - Ctrl+F 本章 / Ctrl+Shift+F 全项目 / Esc 关闭
 * - Enter 循环、Shift+Enter 上一个、跳转不选中文字
 *
 * 契约（老板偏好）：
 * - 输入只显示计数徽章 3/12，不选中文字
 * - 命中高亮 .search-hit / .search-hit-active
 * - 关闭 / 清空搜索时清除所有高亮
 */
import { ref, type Ref } from 'vue'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { searchApi } from '../core/api'
import { nextTick } from 'vue'

export interface ProjectMatch {
  docId: number
  title: string
  line: number
  col: number
  snippet: string
  occurrenceIndex: number
}

interface SearchHighlightState {
  query: string
  activeOccurrence: number
  caseSensitive: boolean
}

export const searchPluginKey = new PluginKey('xzjSearchHighlight')

export function createSearchHighlightPlugin() {
  return new Plugin({
    key: searchPluginKey,
    state: {
      init(): SearchHighlightState {
        return { query: '', activeOccurrence: -1, caseSensitive: false }
      },
      apply(tr, state: SearchHighlightState): SearchHighlightState {
        const meta = tr.getMeta(searchPluginKey)
        if (meta) return { ...state, ...meta }
        return state
      },
    },
    props: {
      decorations(state) {
        const pluginState: SearchHighlightState = this.getState(state) as any
        if (!pluginState || !pluginState.query) return DecorationSet.empty
        const needle = pluginState.caseSensitive ? pluginState.query : pluginState.query.toLowerCase()
        if (!needle) return DecorationSet.empty
        const decos: Decoration[] = []
        let occurrence = 0
        state.doc.descendants((node: any, pos: number) => {
          if (!node.isText) return true
          const text = pluginState.caseSensitive ? (node.text || '') : (node.text || '').toLowerCase()
          let idx = 0
          while (idx < text.length) {
            const found = text.indexOf(needle, idx)
            if (found < 0) break
            const from = pos + found
            const to = from + needle.length
            const isActive = occurrence === pluginState.activeOccurrence
            decos.push(Decoration.inline(from, to, {
              class: isActive ? 'search-hit search-hit-active' : 'search-hit',
            }))
            occurrence++
            idx = found + needle.length
          }
          return true
        })
        return DecorationSet.create(state.doc, decos)
      },
    },
  })
}

interface UseEditorSearchOptions {
  editor: Ref<any>
  projectId: number
  activeDocId: Ref<number | null>
  documents: Ref<any[]>
  openDoc: (doc: any, opts?: { keepSearch?: boolean }) => Promise<void>
}

export function useEditorSearch(opts: UseEditorSearchOptions) {
  const { editor, projectId, activeDocId, documents, openDoc } = opts

  const searchVisible = ref(false)
  const searchScope = ref<'chapter' | 'project'>('chapter')
  const searchQuery = ref('')
  const searchTotal = ref(0)
  const searchActiveIndex = ref(0)
  const searchLoading = ref(false)
  const searchProjectMatches = ref<ProjectMatch[]>([])

  let searchProjectTimer: any = null
  let searchProjectSeq = 0

  function openSearch(scope: 'chapter' | 'project' = 'chapter') {
    // 如果面板已开且模式不同，切模式；否则打开
    if (searchVisible.value && searchScope.value !== scope) {
      onSearchScopeChange(scope)
      return
    }
    searchScope.value = scope
    searchVisible.value = true
  }

  function closeSearch() {
    searchVisible.value = false
    searchQuery.value = ''
    searchTotal.value = 0
    searchActiveIndex.value = 0
    searchProjectMatches.value = []
    searchLoading.value = false
    if (searchProjectTimer) { clearTimeout(searchProjectTimer); searchProjectTimer = null }
    applyEditorSearch('', -1)
  }

  // 数当前编辑器里 query 出现次数（本章模式用）
  function countMatches(query: string): number {
    if (!editor.value || !query) return 0
    const needle = query.toLowerCase()
    let count = 0
    editor.value.state.doc.descendants((node: any) => {
      if (!node.isText) return true
      const text = (node.text || '').toLowerCase()
      let idx = 0
      while (idx < text.length) {
        const found = text.indexOf(needle, idx)
        if (found < 0) break
        count++
        idx = found + needle.length
      }
      return true
    })
    return count
  }

  function onSearchQueryUpdate(q: string) {
    searchQuery.value = q
    if (searchScope.value === 'chapter') {
      const total = countMatches(q)
      searchTotal.value = total
      searchActiveIndex.value = 0
      applyEditorSearch(q, total > 0 ? 0 : -1)
      scrollActiveHitIntoView()
    } else {
      // 全项目：debounce 后端搜索
      if (searchProjectTimer) clearTimeout(searchProjectTimer)
      if (!q.trim()) {
        searchProjectMatches.value = []
        searchTotal.value = 0
        searchActiveIndex.value = 0
        searchLoading.value = false
        applyEditorSearch('', -1)
        return
      }
      searchProjectTimer = setTimeout(() => runProjectSearch(q.trim()), 250)
    }
  }

  function onSearchScopeChange(s: 'chapter' | 'project') {
    searchScope.value = s
    // 重置计数/结果，但保留 query 让用户切模式后立刻能看到结果
    searchActiveIndex.value = 0
    if (s === 'chapter') {
      searchProjectMatches.value = []
      searchLoading.value = false
      const total = countMatches(searchQuery.value)
      searchTotal.value = total
      applyEditorSearch(searchQuery.value, total > 0 ? 0 : -1)
      scrollActiveHitIntoView()
    } else {
      searchTotal.value = 0
      // 全项目模式下编辑器高亮暂时保留 query 描当前文档命中（可选），也可以清掉
      applyEditorSearch(searchQuery.value, -1)
      if (searchQuery.value.trim()) runProjectSearch(searchQuery.value.trim())
    }
  }

  async function runProjectSearch(q: string) {
    const seq = ++searchProjectSeq
    searchLoading.value = true
    try {
      const res = await searchApi.documents(projectId, q, 200)
      if (seq !== searchProjectSeq) return  // 过期请求丢弃
      const data = res.data?.data || res.data
      const items = (data?.items || []) as any[]
      const list: ProjectMatch[] = items.map((it: any) => ({
        docId: it.doc_id,
        title: it.title,
        line: it.line,
        col: it.col,
        snippet: it.snippet,
        occurrenceIndex: it.occurrence_index,
      }))
      searchProjectMatches.value = list
      searchTotal.value = list.length
      // 激活项：优先当前文档内的第一个
      let idx = 0
      if (activeDocId.value) {
        const found = list.findIndex(m => m.docId === activeDocId.value)
        if (found >= 0) idx = found
      }
      searchActiveIndex.value = idx
      // 当前文档内的项：把编辑器高亮同步
      updateProjectHighlightForCurrentDoc()
    } catch (e: any) {
      if (seq !== searchProjectSeq) return
      console.error('全项目搜索失败', e)
      searchProjectMatches.value = []
      searchTotal.value = 0
    } finally {
      if (seq === searchProjectSeq) searchLoading.value = false
    }
  }

  function updateProjectHighlightForCurrentDoc() {
    if (searchScope.value !== 'project') return
    const active = searchProjectMatches.value[searchActiveIndex.value]
    let activeOccurrence = -1
    if (active && active.docId === activeDocId.value) {
      activeOccurrence = active.occurrenceIndex
    }
    applyEditorSearch(searchQuery.value, activeOccurrence)
  }

  function searchNext() {
    if (searchTotal.value === 0) return
    searchActiveIndex.value = (searchActiveIndex.value + 1) % searchTotal.value
    if (searchScope.value === 'chapter') {
      applyEditorSearch(searchQuery.value, searchActiveIndex.value)
      scrollActiveHitIntoView()
    } else {
      onJumpProject(searchProjectMatches.value[searchActiveIndex.value])
    }
  }

  function searchPrev() {
    if (searchTotal.value === 0) return
    searchActiveIndex.value = (searchActiveIndex.value - 1 + searchTotal.value) % searchTotal.value
    if (searchScope.value === 'chapter') {
      applyEditorSearch(searchQuery.value, searchActiveIndex.value)
      scrollActiveHitIntoView()
    } else {
      onJumpProject(searchProjectMatches.value[searchActiveIndex.value])
    }
  }

  async function onJumpProject(m: ProjectMatch) {
    if (!m) return
    const idx = searchProjectMatches.value.findIndex(x => x.docId === m.docId && x.occurrenceIndex === m.occurrenceIndex)
    if (idx >= 0) searchActiveIndex.value = idx
    if (m.docId !== activeDocId.value) {
      const target = documents.value.find(d => d.id === m.docId)
      if (target) {
        await openDoc(target, { keepSearch: true })
        // 等 setContent 的 isSettingContent 完成（openDoc 里 100ms）
        await new Promise(r => setTimeout(r, 140))
      }
    }
    updateProjectHighlightForCurrentDoc()
    scrollActiveHitIntoView()
  }

  function applyEditorSearch(query: string, activeOccurrence: number) {
    if (!editor.value) return
    const view = editor.value.view
    const tr = view.state.tr.setMeta(searchPluginKey, {
      query,
      activeOccurrence,
      caseSensitive: false,
    })
    view.dispatch(tr)
  }

  function scrollActiveHitIntoView() {
    nextTick(() => {
      const el = document.querySelector('.ProseMirror .search-hit-active') as HTMLElement | null
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    })
  }

  return {
    // state
    searchVisible,
    searchScope,
    searchQuery,
    searchTotal,
    searchActiveIndex,
    searchLoading,
    searchProjectMatches,
    // methods
    openSearch,
    closeSearch,
    onSearchQueryUpdate,
    onSearchScopeChange,
    searchNext,
    searchPrev,
    onJumpProject,
    applyEditorSearch,
    updateProjectHighlightForCurrentDoc,
  }
}
