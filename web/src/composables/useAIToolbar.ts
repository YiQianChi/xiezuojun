/**
 * useAIToolbar — AI 续写/改写/润色/扩写工具栏 composable
 *
 * 从 Editor.vue 抽出的 AI 工具栏逻辑：
 * - AI 高亮 / 删除线 Marks（AIMark / AIStrikethrough）供 tiptap extensions 使用
 * - 选区工具栏（选中文字浮出 [续写/改写/润色/扩写] 按钮）
 * - 方向输入中间步骤（用户可输入改写方向，也可不输直接回车）
 * - 插入正文后浮动预览工具栏（[停止/采用/丢弃/重新生成]）
 * - 交互模式：原文删除线 + AI 结果绿高亮 → 采用/丢弃/重新生成（老板明确偏好）
 *
 * 契约（老板偏好）：
 * - 采用：非续写模式真正删除原文，AI 变正文
 * - 丢弃：删掉 AI 文本，去掉原文删除线
 * - 重新生成：先丢弃 → 回到方向输入框（inputting 状态）
 *
 * 状态机：
 *   idle →（点按钮）→ inputting →（回车/生成）→ generating →（流结束）→ preview
 *     preview →（采用/丢弃）→ idle
 *     preview →（重新生成）→ inputting
 *     generating →（停止）→ preview 或 idle
 */
import { ref, nextTick, type Ref } from 'vue'
import { Mark } from '@tiptap/core'
import { streamGenerate } from '../core/sse'
import { toast } from './useToast'

export type AIAction = 'continue' | 'rewrite' | 'polish' | 'expand'

/** AI 交互状态机 */
export type AIPhase = 'idle' | 'inputting' | 'generating' | 'preview'

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
  usedSettings: Ref<UsedSetting[]>
  saveStatus: Ref<string>
}

export function useAIToolbar(opts: UseAIToolbarOptions) {
  const { editor, projectId, activeDocId, usedSettings, saveStatus } = opts

  const aiLoading = ref(false)

  /** AI 交互状态机当前阶段 */
  const aiPhase = ref<AIPhase>('idle')
  /** 用户输入的改写方向（可空） */
  const aiDirection = ref('')
  /** 流式生成可中断控制器 */
  let abortController: AbortController | null = null
  /** prepareAIAction 记录、executeAIAction 消用的 action */
  const pendingAction = ref<AIAction | null>(null)

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

  function buildAIPrompt(action: AIAction, selectedText: string, direction?: string) {
    const instructions: Record<AIAction, string> = {
      continue: '你是小说写作助手。请直接续写下面的小说文本，只输出续写的正文内容。不要解释、不要加标题、不要输出多个版本、不要分析。直接开始写小说正文。',
      rewrite: '你是小说编辑。请直接改写下面的小说文本，保留核心情节，提升表达。只输出改写后的正文，不要解释、不要分析、不要给多个版本。直接输出改写后的最终文本。',
      polish: '你是文字编辑。请直接润色下面的小说文本，保持原意，修正语病，提升文字质感。只输出润色后的正文，不要解释、不要分析、不要给多个版本。直接输出最终文本。',
      expand: '你是小说作家。请直接扩写下面的小说文本，增加细节、氛围、动作和心理描写。只输出扩写后的正文，不要解释、不要分析、不要给多个版本。直接输出最终文本。',
    }
    let prompt = `${instructions[action]}\n\n${selectedText}`
    if (direction && direction.trim()) {
      prompt += `\n\n改写方向：${direction.trim()}`
    }
    return prompt
  }

  function cleanAIResult(text: string) {
    return String(text || '')
      .replace(/^```[\s\S]*?\n/, '')
      .replace(/```$/g, '')
      .replace(/\n{2,}/g, '\n') // ponytail: AI 输出 \n\n 段间空行 → 归一化为单换行
      .trim()
  }

  function applyTextMark(markName: string, from: number, to: number, attrs: Record<string, any> = {}) {
    if (!editor.value || from >= to) return
    const { state, view } = editor.value
    const markType = state.schema.marks[markName]
    if (!markType) return
    // ponytail: ProseMirror tr.addMark already handles node boundaries correctly.
    // Manual nodesBetween loop had an off-by-one at text node start (first char unmarked).
    const tr = state.tr.addMark(from, to, markType.create(attrs))
    if (tr.docChanged) view.dispatch(tr)
  }

  function removeTextMark(markName: string, from: number, to: number) {
    if (!editor.value || from >= to) return
    const { state, view } = editor.value
    const markType = state.schema.marks[markName]
    if (!markType) return
    const tr = state.tr.removeMark(from, to, markType)
    if (tr.docChanged) view.dispatch(tr)
  }

  // ---- 第一阶段：点工具栏按钮 → 进入方向输入态 ----
  function prepareAIAction(action: AIAction) {
    if (!editor.value || !aiToolbar.value.text || aiLoading.value) return
    // 如果上一次 AI 插入还在等待决定，先忽略新请求
    if (aiInsert.value.active) return

    const isContinue = action === 'continue'

    // 非续写模式：先给原文加删除线
    if (!isContinue) {
      const { start, end } = aiToolbar.value
      applyTextMark('aiStrikethrough', start, end)
    }

    // 隐藏浮动工具栏，进入方向输入态
    aiToolbar.value.visible = false
    aiPhase.value = 'inputting'
    aiDirection.value = ''

    // 记录当前 action 供 execute 使用
    pendingAction.value = action
  }

  // ---- 第二阶段：用户输入方向后点"生成"或回车 → 流式生成 ----
  async function executeAIAction() {
    if (aiPhase.value !== 'inputting' || !pendingAction.value) return
    if (!editor.value || !aiToolbar.value.text) return

    const action = pendingAction.value
    aiPhase.value = 'generating'
    aiLoading.value = true
    const oldStatus = saveStatus.value
    saveStatus.value = '生成中...'

    const { start, end, text: selectedText } = aiToolbar.value

    // ---- 准备插入位置 ----
    // 续写模式：原文不加删除线；非续写模式已在 prepareAIAction 加过删除线
    const insertPos = end
    const originalRange = { from: start, to: end }

    // 先插入一个换行，作为 AI 文本的起始锚点
    editor.value.chain()
      .focus()
      .setTextSelection({ from: insertPos, to: insertPos })
      .insertContent('\n')
      .run()

    const aiFrom = insertPos + 1  // AI 文本起始位置（换行之后）

    // 记住原文信息
    aiInsert.value = {
      active: true,
      x: 0, y: 0,
      from: aiFrom,
      to: aiFrom,
      action,
      originalText: selectedText,
      originalFrom: originalRange.from,
      originalTo: originalRange.to,
    }

    // ---- 流式接收 + rAF 节流写入编辑器 ----
    let accumulated = ''
    let pendingFlush = false

    function flushToEditor() {
      pendingFlush = false
      if (!editor.value) return
      const cleaned = cleanAIResult(accumulated)
      if (!cleaned) return

      // 替换 aiFrom → aiInsert.value.to 之间的内容为最新全文
      const replaceFrom = aiFrom
      const replaceTo = aiInsert.value.to
      const insertText = cleaned

      editor.value.chain()
        .deleteRange({ from: replaceFrom, to: replaceTo })
        .insertContentAt({ from: replaceFrom, to: replaceFrom }, insertText)
        .run()

      aiInsert.value.to = replaceFrom + insertText.length

      // 插入后清除继承的删除线，再加绿色高亮
      removeTextMark('aiStrikethrough', aiFrom, aiInsert.value.to)
      applyTextMark('aiHighlight', aiFrom, aiInsert.value.to, { action })

      // 光标跟到末尾
      editor.value.commands.setTextSelection(aiInsert.value.to)
    }

    abortController = new AbortController()

    try {
      await streamGenerate(
        {
          project_id: String(projectId),
          document_id: activeDocId.value ? String(activeDocId.value) : undefined,
          mode: action,
          content: buildAIPrompt(action, selectedText, aiDirection.value),
        },
        {
          onSettings: (settings) => {
            usedSettings.value = settings.map((s: any) => ({
              id: s.id, title: s.title, type: s.type,
              used: !!s.used, tokens: s.tokens || 0,
            }))
          },
          onContent: (chunk) => {
            accumulated += chunk
            if (!pendingFlush) {
              pendingFlush = true
              requestAnimationFrame(flushToEditor)
            }
          },
        },
        abortController.signal,
      )

      // 流结束：最终 flush 一次确保完整渲染
      flushToEditor()

      const result = cleanAIResult(accumulated)
      if (!result) throw new Error('AI 返回为空')

      aiPhase.value = 'preview'

      // 计算浮动工具栏位置
      nextTick(() => {
        if (!editor.value) return
        const coords = editor.value.view.coordsAtPos(aiInsert.value.to)
        aiInsert.value.x = coords.left + 10
        aiInsert.value.y = coords.bottom + 8
      })

      saveStatus.value = '已生成，等待采用'
    } catch (e: any) {
      console.error('AI 生成失败', e)
      saveStatus.value = oldStatus || '生成失败'
      // abort 引发的错误不弹 toast
      if (!(e?.name === 'AbortError')) {
        toast.error(e.message || '生成失败')
      }
      // 如果已经有部分内容，进入 preview；否则清理状态回到 idle
      if (accumulated && aiInsert.value.active && aiInsert.value.to > aiInsert.value.from) {
        aiPhase.value = 'preview'
      } else {
        aiPhase.value = 'idle'
        aiInsert.value.active = false
      }
    } finally {
      aiLoading.value = false
      abortController = null
    }
  }

  // ---- 停止生成 ----
  function stopAIGeneration() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    aiLoading.value = false
    // 如果已有部分内容，进入 preview 状态；否则回到 idle
    if (aiInsert.value.active && aiInsert.value.to > aiInsert.value.from) {
      aiPhase.value = 'preview'
    } else {
      aiPhase.value = 'idle'
      aiInsert.value.active = false
    }
    saveStatus.value = '已停止'
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
    aiPhase.value = 'idle'
    pendingAction.value = null
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
    aiPhase.value = 'idle'
    pendingAction.value = null
    saveStatus.value = '编辑中...'
  }

  // ---- 重新生成：丢弃 → 回到方向输入 ----
  async function regenerateAIResult() {
    if (!editor.value || !aiInsert.value.active) return
    // 先丢弃当前结果（pendingAction 被 discard 清掉，这里恢复）
    const action = aiInsert.value.action as AIAction
    discardAIResult()
    // 回到方向输入态
    aiPhase.value = 'inputting'
    aiDirection.value = ''
    pendingAction.value = action
  }

  // ---- 取消方向输入（从 inputting 退回 idle，清理删除线）----
  function cancelInputting() {
    if (aiPhase.value !== 'inputting') return
    // 非续写模式：去掉原文删除线
    if (pendingAction.value && pendingAction.value !== 'continue') {
      const { start, end } = aiToolbar.value
      removeTextMark('aiStrikethrough', start, end)
    }
    aiPhase.value = 'idle'
    pendingAction.value = null
    aiDirection.value = ''
  }

  // ---- 审核模式兼容入口：直接改写，跳过方向输入 ----
  async function quickRewrite(text: string, start: number, end: number) {
    aiToolbar.value = {
      visible: false,
      x: 0, y: 0,
      text,
      start,
      end,
      action: 'rewrite',
      result: '',
    }
    pendingAction.value = 'rewrite'
    aiDirection.value = ''
    aiPhase.value = 'inputting'
    await executeAIAction()
  }

  return {
    // state
    aiLoading,
    aiPhase,
    aiDirection,
    aiToolbar,
    aiInsert,
    // methods
    pendingAction,
    showAIToolbar,
    prepareAIAction,
    executeAIAction,
    stopAIGeneration,
    acceptAIResult,
    discardAIResult,
    regenerateAIResult,
    cancelInputting,
    quickRewrite,
  }
}
