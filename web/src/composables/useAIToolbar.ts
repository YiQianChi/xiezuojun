/**
 * useAIToolbar — AI 续写/改写/润色/扩写工具栏 composable
 *
 * 从 Editor.vue 抽出的 AI 工具栏逻辑：
 * - AI 高亮 / 删除线 Marks（AIMark / AIStrikethrough）供 tiptap extensions 使用
 * - 选区工具栏（选中文字浮出 [续写/改写/润色/扩写] 按钮）
 * - 插入正文后浮动预览工具栏（[重新生成/丢弃/采用]）
 * - 交互模式：原文删除线 + AI 结果绿高亮 → 采用/丢弃/重新生成（老板明确偏好）
 *
 * 契约（老板偏好）：
 * - 采用：非续写模式真正删除原文，AI 变正文
 * - 丢弃：删掉 AI 文本，去掉原文删除线
 * - 重新生成：先丢弃 → 再跑一次
 */
import { ref, nextTick, type Ref } from 'vue'
import { Mark } from '@tiptap/core'
import { aiApi } from '../core/api'

export type AIAction = 'continue' | 'rewrite' | 'polish' | 'expand'

// AI 高亮 Mark — 标记 AI 生成文本（绿色背景）
export const AIMark = Mark.create({
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
export const AIStrikethrough = Mark.create({
  name: 'aiStrikethrough',
  parseHTML() {
    return [{ tag: 'span[data-ai-strike]' }]
  },
  renderHTML() {
    return ['span', { 'data-ai-strike': '', class: 'ai-strike-mark' }, 0]
  },
})

interface UsedSetting {
  id: number
  title: string
  type: string
  used: boolean
  tokens: number
}

interface UseAIToolbarOptions {
  editor: Ref<any>
  projectId: number
  activeDocId: Ref<number | null>
  includeSettings: Ref<boolean>
  usedSettings: Ref<UsedSetting[]>
  saveStatus: Ref<string>
}

export function useAIToolbar(opts: UseAIToolbarOptions) {
  const { editor, projectId, activeDocId, includeSettings, usedSettings, saveStatus } = opts

  const aiLoading = ref(false)

  const aiToolbar = ref({
    visible: false,
    x: 0, y: 0,
    text: '',
    start: 0,
    end: 0,
    action: '' as AIAction | '',
    result: '',
  })

  // AI 结果已插入正文后的浮动工具栏状态
  const aiInsert = ref({
    active: false,   // 是否有 AI 文本在等待用户决定
    x: 0, y: 0,      // 浮动工具栏位置
    from: 0, to: 0,  // AI 文本在文档中的位置
    action: '' as AIAction | '',
    originalText: '',  // 选中的原文（丢弃时恢复）
    originalFrom: 0, originalTo: 0, // 原文的位置
  })

  function showAIToolbar(sel: { text: string; start: number; end: number }) {
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
      result: '',
    }
  }

  function hideAIToolbar() {
    aiToolbar.value.visible = false
  }

  function resetAIState() {
    aiToolbar.value.visible = false
    aiInsert.value.active = false
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

  return {
    // state
    aiLoading,
    aiToolbar,
    aiInsert,
    // methods
    showAIToolbar,
    hideAIToolbar,
    resetAIState,
    runAIAction,
    acceptAIResult,
    discardAIResult,
    regenerateAIResult,
  }
}
