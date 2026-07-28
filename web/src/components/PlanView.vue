<template>
  <div class="plan-view">
    <div class="plan-inner">
      <h2>《{{ projectName || '未命名项目' }}》· 构思阶段</h2>
      <div class="subtitle">跟着步骤走，或者跳过任意一步。AI 每步的产物你都能改，不满意重新生成。</div>

      <div class="step-list">
        <div
          v-for="step in steps"
          :key="step.id"
          class="step-card"
          :class="{ 'is-doing': step.status === 'doing' }"
        >
          <div class="step-head">
            <span class="step-num">{{ step.id }}</span>
            <span class="step-title">{{ step.title }}</span>
            <span class="step-status" :class="step.status">{{ statusLabel(step.status) }}</span>
          </div>
          <div class="step-body">{{ step.desc }}</div>

          <textarea
            v-if="step.status !== 'doing'"
            v-model="step.userNote"
            class="step-note-input"
            rows="3"
            :placeholder="step.promptHint"
          ></textarea>
          <div v-if="step.status === 'pending' && !step.userNote?.trim()" class="step-write-hint">✍️ 先写点内容，再让 AI 帮你完善</div>

          <!-- AI 生成结果，可展开编辑 -->
          <div v-if="step.preview != null" class="step-preview-wrap">
            <textarea
              v-if="step.editing"
              v-model="step.preview"
              class="step-editor"
              rows="10"
            ></textarea>
            <div v-else class="step-preview" @click="step.editing = true">
              <template v-if="step.status === 'doing' && !step.preview">
                <span class="step-loading-spinner"></span> AI 思考中…
              </template>
              <template v-else>{{ step.preview }}</template>
            </div>
          </div>

          <div v-if="step.error" class="step-error">{{ step.error }}</div>

          <div class="step-actions">
            <button v-if="step.status === 'done'" @click="startStep(step.id)">重新生成</button>
            <button v-if="step.status === 'done' && !step.editing" @click="step.editing = true">编辑</button>
            <button v-if="step.status === 'done' && step.editing" @click="step.editing = false">完成编辑</button>
            <button v-if="step.status === 'done'" class="primary" :disabled="step.writing" @click="openFolderPicker(step)">
              {{ step.writing ? '写入中...' : '写入设定' }}
            </button>
            <button v-if="step.status === 'doing'" disabled>AI 生成中...</button>
            <button v-if="step.status === 'pending'" class="primary" :disabled="!step.userNote?.trim()" @click="startStep(step.id)">让 AI 完善</button>
          </div>
        </div>
      </div>

      <div class="plan-hint">
        💡 <strong>提示：</strong>构思模式的所有产出都会写入左侧 <strong>「设定」Tab</strong>。写作时 AI 会自动读取这些设定作为背景。想改设定内容，切到写作模式后在设定 Tab 里直接编辑就行。
      </div>
    </div>

    <!-- 文件夹选择弹窗 -->
    <div v-if="showFolderPicker" class="folder-picker-overlay" @click.self="closeFolderPicker">
      <div class="folder-picker">
        <div class="fp-header">
          <span class="fp-title">选择保存位置</span>
          <button class="fp-close" @click="closeFolderPicker">✕</button>
        </div>
        <div class="fp-body">
          <div class="fp-field">
            <label>文件名</label>
            <input v-model="pickerFileName" type="text" placeholder="输入文件名" />
          </div>
          <div class="fp-field">
            <label>目标文件夹</label>
            <div v-if="pickerFolders.length === 0" class="fp-empty">暂无设定文件夹，请先初始化项目结构</div>
            <div v-else class="fp-folder-list">
              <label
                v-for="f in pickerFolders"
                :key="f.id"
                class="fp-folder-item"
                :class="{ active: pickerFolderId === f.id }"
              >
                <input type="radio" v-model="pickerFolderId" :value="f.id" />
                <span class="fp-icon">📁</span>
                <span class="fp-name">{{ f.title }}</span>
              </label>
            </div>
          </div>
          <div v-if="pickerError" class="fp-error">{{ pickerError }}</div>
        </div>
        <div class="fp-footer">
          <button class="fp-cancel" @click="closeFolderPicker">取消</button>
          <button class="fp-confirm" :disabled="!pickerFolderId || pickerWriting" @click="confirmWrite">
            {{ pickerWriting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { documentApi } from '../core/api'
import { streamGenerate } from '../core/sse'

const props = defineProps<{
  projectName: string
  projectId: number
}>()

const emit = defineEmits<{
  (e: 'jump-to-doc', docId: number): void
}>()

interface PlanStep {
  id: number
  title: string
  desc: string
  status: 'pending' | 'doing' | 'done'
  preview?: string
  userNote?: string
  promptHint: string
  editing?: boolean
  error?: string
  writing?: boolean
  settingType: 'outline' | 'characters' | 'worldbuilding'
  prompt: string
}

const steps = ref<PlanStep[]>([
  {
    id: 1,
    title: '故事梗概',
    desc: '用一段话讲清楚：什么人 / 在什么背景 / 遇到什么事 / 怎么改变。',
    status: 'pending',
    settingType: 'outline',
    prompt: '请为这部小说写一段 300-500 字的故事梗概。包含：主角是谁、所处的背景、遇到什么事件、如何转变。要有钩子，让人想继续读。',
    promptHint: '写下你的故事梗概：什么人、在什么背景、遇到什么事、怎么改变……',
  },
  {
    id: 2,
    title: '世界观 · 关键设定',
    desc: '时代 / 地点 / 规则 / 冲突源头。AI 会生成多张世界观卡。',
    status: 'pending',
    settingType: 'worldbuilding',
    prompt: '【必须基于上一步确定的故事梗概】请构建能支撑这个故事的世界观设定，包括：时代背景、地理环境、势力划分、能力体系（如有）、规则与禁忌、冲突的根源。世界观中的每一个设定都要服务于梗概中已确定的故事。用 Markdown 分段输出。',
    promptHint: '写下你的世界观：什么时代、什么地点、有什么规则、冲突从哪来……',
  },
  {
    id: 3,
    title: '人物关系',
    desc: '主要人物 3-5 人，各自的身份、动机、秘密。',
    status: 'pending',
    settingType: 'characters',
    prompt: '【必须基于已确定的故事梗概和世界观】请设计服务于这个故事的主要人物（3-5人）。人物必须生活在已确定的世界观设定中，其身份、动机、秘密都要与梗概中的核心冲突直接相关。每个人物包括：姓名、身份、性格特点、核心动机、不为人知的秘密。最后附上人物之间的关系。用 Markdown 格式输出。',
    promptHint: '写下你的人物：名字、身份、性格、想要什么、藏着什么秘密……',
  },
  {
    id: 4,
    title: '故事主线 · 三幕结构',
    desc: '开端 / 冲突升级 / 高潮与结局。',
    status: 'pending',
    settingType: 'outline',
    prompt: '【必须基于已确定的梗概、世界观和人物】请用三幕结构设计故事主线，把已确定的人物放进已确定的世界观中，推动梗概中的核心冲突走向结局。第一幕（开端/日常→打破日常的契机）、第二幕（冲突升级/障碍与成长）、第三幕（高潮与结局）。每幕列出关键情节点，并标注涉及的核心人物。用 Markdown 格式输出。',
    promptHint: '写下你的故事主线：怎么开始、冲突怎么升级、高潮和结局是什么……',
  },
  {
    id: 5,
    title: '章节大纲 · 目录',
    desc: '按主线切分卷 / 章，每章一句话。完成后可切到「写作」模式。',
    status: 'pending',
    settingType: 'outline',
    prompt: '【必须基于以上全部设定：梗概、世界观、人物、三幕结构】请将三幕结构主线切分为章节大纲。按「卷→章」结构，每章给一句话标题和一句概要。章节顺序必须遵循三幕结构的节奏，涉及的人物必须来自已确定的人物列表。用 Markdown 格式输出。',
    promptHint: '列出你想写的章节：每章标题和一句话概要……',
  },
])

// ---- 本地草稿持久化（刷新不丢） ----
const DRAFT_KEY = `plan-draft-${props.projectId}`

interface DraftEntry {
  status: string
  preview?: string
  userNote?: string
}

function loadDraft() {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) return
    const saved: Record<number, DraftEntry> = JSON.parse(raw)
    for (const s of steps.value) {
      const d = saved[s.id]
      if (!d) continue
      // doing 是瞬态（生成中途刷新），恢复成 pending 让用户重新点
      if (d.status === 'doing') continue
      s.status = d.status as PlanStep['status']
      s.preview = d.preview
      s.userNote = d.userNote
    }
  } catch { /* 旧格式/损坏，忽略 */ }
}

// 只持久化需要保留的字段，排除 prompt/desc 等静态定义（代码改了不会冲突）
function saveDraft() {
  const data: Record<number, DraftEntry> = {}
  for (const s of steps.value) {
    if (s.status === 'pending' && !s.userNote?.trim()) continue
    data[s.id] = { status: s.status, preview: s.preview, userNote: s.userNote }
  }
  localStorage.setItem(DRAFT_KEY, JSON.stringify(data))
}

watch(steps, saveDraft, { deep: true })
loadDraft()

function statusLabel(s: string) {
  return s === 'done' ? '已完成' : s === 'doing' ? '进行中' : '未开始'
}

async function startStep(id: number) {
  const step = steps.value.find(s => s.id === id)
  if (!step || step.status === 'doing') return

  step.status = 'doing'
  step.error = undefined
  step.preview = ''  // 立刻显示渲染框

  try {
    // 收集已完成步骤的结果作为上下文
    const contextParts: string[] = []
    // 书名只在第一步作为锚点，之后弱化为辅助信息
    if (step.id === 1) {
      contextParts.push(`项目名：${props.projectName || '未命名'}（注意：书名可能一语双关，请结合用户的构思判断方向，不要仅凭书名猜测）`)
    } else {
      contextParts.push(`项目名：${props.projectName || '未命名'}（仅供识别，以下设定才是准确依据）`)
    }
    if (step.userNote?.trim()) {
      contextParts.push(`作者补充：${step.userNote.trim()}`)
    }
    const doneSteps = steps.value.filter(s => s.status === 'done' && s.preview)
    if (doneSteps.length > 0) {
      contextParts.push('以下是前面已确定的核心设定，你必须严格继承，不得与之矛盾：')
      doneSteps.forEach(s => {
        contextParts.push(`【${s.title}】\n${s.preview}`)
      })
      contextParts.push('以上设定为约束条件，不是参考。新内容必须与它们保持完全一致。')
    }

    let accumulated = ''
    let rafId = 0
    await streamGenerate(
      {
        project_id: String(props.projectId),
        mode: 'plan',
        content: `${step.prompt}\n\n作者原始构思：\n${step.userNote || '（无补充，请基于已有设定生成）'}\n\n请基于作者的构思进行完善和扩展，保留核心创意，补充细节、逻辑和结构。输出完整的设定内容。\n\n${contextParts.join('\n\n')}`,
      },
      {
        onContent: (chunk) => {
          accumulated += chunk
          // ponytail: 首个chunk直接赋值（立刻消失spinner），后续rAF节流
          if (!step.preview) {
            step.preview = accumulated
          } else if (!rafId) {
            rafId = requestAnimationFrame(() => {
              step.preview = accumulated
              rafId = 0
            })
          }
        },
      },
    )

    const result = accumulated.trim()
    if (!result) throw new Error('AI 返回为空')

    if (rafId) cancelAnimationFrame(rafId)
    step.preview = accumulated.trim() || result
    step.status = 'done'
    step.editing = false
  } catch (e: any) {
    step.status = 'pending'
    step.error = e.message || 'AI 生成失败，请重试'
  }
}

// ---- 文件夹选择弹窗 ----
const showFolderPicker = ref(false)
const pickerFolders = ref<any[]>([])
const pickerFolderId = ref<number | null>(null)
const pickerFileName = ref('')
const pickerWriting = ref(false)
const pickerError = ref('')
const pickerStep = ref<PlanStep | null>(null)

const SETTING_TYPES = ['outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline']

async function openFolderPicker(step: PlanStep) {
  pickerStep.value = step
  pickerFileName.value = step.title
  pickerFolderId.value = null
  pickerError.value = ''
  pickerFolders.value = []
  showFolderPicker.value = true
  try {
    const res = await documentApi.list(String(props.projectId))
    const docs = res.data?.data || res.data || []
    pickerFolders.value = docs.filter((d: any) => SETTING_TYPES.includes(d.type) && !d.parent_id)
    // 默认选当前 step 推荐 type 对应的文件夹
    const preferred = pickerFolders.value.find((d: any) => d.type === step.settingType)
    if (preferred) pickerFolderId.value = preferred.id
  } catch (e: any) {
    pickerError.value = '加载文件夹失败'
  }
}

function closeFolderPicker() {
  if (pickerWriting.value) return
  showFolderPicker.value = false
  pickerStep.value = null
}

async function confirmWrite() {
  const step = pickerStep.value
  const folderId = pickerFolderId.value
  if (!step || !folderId) return

  const folder = pickerFolders.value.find(f => f.id === folderId)
  if (!folder) return

  pickerWriting.value = true
  pickerError.value = ''
  step.writing = true
  try {
    const content = `# ${pickerFileName.value}\n\n${step.preview}`
    const res = await documentApi.list(String(props.projectId))
    const docs = res.data?.data || res.data || []
    // ponytail: 同名子文档存在则更新，否则新建
    const existing = docs.find((d: any) => d.parent_id === folderId && d.title === pickerFileName.value)
    if (existing) {
      await documentApi.update(existing.id, { content })
      pickerWriting.value = false
      step.writing = false
      showFolderPicker.value = false
      pickerStep.value = null
      emit('jump-to-doc', existing.id)
    } else {
      const createRes = await documentApi.create({
        project_id: Number(props.projectId),
        parent_id: folderId,
        type: 'note',
        title: pickerFileName.value,
        content,
      })
      pickerWriting.value = false
      step.writing = false
      showFolderPicker.value = false
      pickerStep.value = null
      const newId = createRes.data?.data?.id || createRes.data?.id
      if (newId) emit('jump-to-doc', newId)
    }
  } catch (e: any) {
    pickerWriting.value = false
    step.writing = false
    pickerError.value = e.response?.data?.error || '写入失败，请重试'
  }
}
</script>

<style scoped>
.plan-view {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-primary, #fafafa);
}
.plan-inner {
  max-width: 820px;
  margin: 0 auto;
  padding: 32px 40px 60px;
}
.plan-view h2 {
  font-family: var(--font-serif);
  margin: 0 0 6px 0;
  font-size: 22px;
  color: var(--text-primary);
}
.plan-view .subtitle {
  color: var(--text-secondary);
  margin-bottom: 16px;
  font-size: 13px;
}

.step-note-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  outline: none;
  background: var(--bg-secondary, #fff);
  color: var(--text-primary);
  font-family: var(--font-serif);
  box-sizing: border-box;
  margin-bottom: 10px;
}
.step-note-input:focus {
  border-color: #af52de;
  box-shadow: 0 0 0 3px rgba(175, 82, 222, 0.1);
}
.step-note-input::placeholder {
  color: var(--text-tertiary, #aaa);
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.step-card {
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  border: 1px solid var(--border);
  padding: 18px 20px;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.15s ease;
}
.step-card.is-doing {
  box-shadow: 0 4px 20px rgba(175, 82, 222, 0.1);
  border-color: rgba(175, 82, 222, 0.2);
}

.step-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.step-num {
  width: 26px; height: 26px;
  border-radius: 50%;
  background: rgba(175, 82, 222, 0.12);
  color: #af52de;
  font-size: 13px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.step-title { font-weight: 600; font-size: 15px; flex: 1; color: var(--text-primary); }

.step-status {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
}
.step-status.done { background: rgba(52, 199, 89, 0.12); color: #34c759; }
.step-status.pending { background: var(--segment-bg); color: var(--text-secondary); }
.step-status.doing { background: rgba(0, 122, 255, 0.12); color: var(--accent, #007aff); }

.step-body {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
  margin-bottom: 10px;
}

.step-preview-wrap {
  margin-bottom: 10px;
}
.step-preview {
  background: var(--hover-bg);
  border-radius: 8px;
  padding: 12px 14px;
  font-family: var(--font-serif);
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.7;
  white-space: pre-wrap;
  cursor: text;
  border: 1px solid transparent;
}
.step-preview:hover {
  border-color: rgba(175, 82, 222, 0.2);
}
.step-loading-spinner {
  display: inline-block;
  width: 16px; height: 16px;
  border: 2px solid rgba(0, 122, 255, 0.2);
  border-top-color: var(--accent, #007aff);
  border-radius: 50%;
  animation: step-spin .8s linear infinite;
  vertical-align: middle;
  margin-right: 6px;
}
@keyframes step-spin { to { transform: rotate(360deg); } }
.step-editor {
  width: 100%;
  background: rgba(175, 82, 222, 0.03);
  border: 1px solid rgba(175, 82, 222, 0.3);
  border-radius: 8px;
  padding: 12px 14px;
  font-family: var(--font-serif);
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.7;
  resize: vertical;
  outline: none;
}

.step-error {
  color: #ff3b30;
  font-size: 12px;
  margin-bottom: 8px;
}

.step-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.step-actions button {
  padding: 5px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 7px;
  background: var(--bg-secondary, #ffffff);
  font-size: 12px;
  cursor: pointer;
  color: var(--text-primary);
}
.step-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.step-actions button.primary {
  background: #af52de;
  color: white;
  border-color: #af52de;
}
.step-actions button.primary:hover:not(:disabled) {
  background: #9740c2;
}

.plan-hint {
  margin-top: 28px;
  padding: 14px 16px;
  background: rgba(175, 82, 222, 0.05);
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);
}
.plan-hint strong { color: var(--text-primary); }

/* 文件夹选择弹窗 */
.folder-picker-overlay {
  position: fixed; inset: 0;
  background: var(--overlay-bg);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000;
}
.folder-picker {
  width: 380px;
  background: var(--bg-secondary, #fff);
  border-radius: 14px;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}
.fp-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border, #e5e5ea);
}
.fp-title { font-weight: 600; font-size: 16px; color: var(--text-primary); }
.fp-close {
  border: none; background: transparent; font-size: 18px;
  cursor: pointer; color: var(--text-secondary); line-height: 1;
}
.fp-body { padding: 16px 20px; }
.fp-field { margin-bottom: 14px; }
.fp-field label {
  display: block; font-size: 12px; font-weight: 600;
  color: var(--text-secondary); margin-bottom: 6px;
}
.fp-field input[type="text"] {
  width: 100%; padding: 8px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px; font-size: 14px;
  outline: none; box-sizing: border-box;
}
.fp-field input[type="text"]:focus {
  border-color: #af52de;
}
.fp-folder-list { display: flex; flex-direction: column; gap: 4px; }
.fp-folder-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; border-radius: 8px; cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.12s;
}
.fp-folder-item:hover { background: var(--hover-bg); }
.fp-folder-item.active {
  background: rgba(175,82,222,0.08);
  border-color: rgba(175,82,222,0.2);
}
.fp-folder-item input { accent-color: #af52de; }
.fp-icon { font-size: 16px; }
.fp-name { font-size: 14px; color: var(--text-primary); }
.fp-empty { font-size: 13px; color: var(--text-tertiary); padding: 8px 0; }
.fp-error { color: #ff3b30; font-size: 12px; margin-top: 4px; }
.fp-footer {
  display: flex; gap: 8px; justify-content: flex-end;
  padding: 12px 20px; border-top: 1px solid var(--border, #e5e5ea);
}
.fp-cancel, .fp-confirm {
  padding: 8px 20px; border-radius: 8px; font-size: 14px; cursor: pointer;
}
.fp-cancel {
  border: 1px solid var(--border, #e5e5ea);
  background: var(--bg-secondary, #fff);
  color: var(--text-primary);
}
.fp-confirm {
  border: none; background: #af52de; color: white; font-weight: 500;
}
.fp-confirm:disabled { opacity: 0.5; cursor: not-allowed; }
.fp-confirm:hover:not(:disabled) { background: #9740c2; }
.step-write-hint { font-size: 12px; color: var(--text-tertiary); margin-top: 4px; }
</style>
