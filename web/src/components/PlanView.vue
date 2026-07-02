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

          <!-- AI 生成结果，可展开编辑 -->
          <div v-if="step.preview" class="step-preview-wrap">
            <textarea
              v-if="step.editing"
              v-model="step.preview"
              class="step-editor"
              rows="10"
            ></textarea>
            <div v-else class="step-preview" @click="step.editing = true">{{ step.preview }}</div>
          </div>

          <div v-if="step.error" class="step-error">{{ step.error }}</div>

          <div class="step-actions">
            <button v-if="step.status === 'done'" @click="startStep(step.id)">重新生成</button>
            <button v-if="step.status === 'done' && !step.editing" @click="step.editing = true">编辑</button>
            <button v-if="step.status === 'done' && step.editing" @click="step.editing = false">完成编辑</button>
            <button v-if="step.status === 'done'" class="primary" :disabled="step.writing" @click="writeToSetting(step)">
              {{ step.writing ? '写入中...' : '写入设定 Tab' }}
            </button>
            <button v-if="step.status === 'doing'" disabled>AI 生成中...</button>
            <button v-if="step.status === 'pending'" class="primary" @click="startStep(step.id)">开始</button>
          </div>
        </div>
      </div>

      <div class="plan-hint">
        💡 <strong>提示：</strong>计划模式的所有产出都会写入左侧 <strong>「设定」Tab</strong>。写作时 AI 会自动读取这些设定作为背景。想改设定内容，切到写作模式后在设定 Tab 里直接编辑就行。
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { aiApi, documentApi } from '../core/api'

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
  },
  {
    id: 2,
    title: '世界观 · 关键设定',
    desc: '时代 / 地点 / 规则 / 冲突源头。AI 会生成多张世界观卡。',
    status: 'pending',
    settingType: 'worldbuilding',
    prompt: '请构建这部小说的世界观设定，包括：时代背景、地理环境、势力划分、能力体系（如有）、规则与禁忌、冲突的根源。用 Markdown 分段输出。',
  },
  {
    id: 3,
    title: '人物关系',
    desc: '主要人物 3-5 人，各自的身份、动机、秘密。',
    status: 'pending',
    settingType: 'characters',
    prompt: '请设计这部小说的主要人物（3-5人）。每个人物包括：姓名、身份、性格特点、核心动机、不为人知的秘密。最后附上人物之间的关系。用 Markdown 格式输出。',
  },
  {
    id: 4,
    title: '故事主线 · 三幕结构',
    desc: '开端 / 冲突升级 / 高潮与结局。',
    status: 'pending',
    settingType: 'outline',
    prompt: '请用三幕结构设计故事主线：第一幕（开端/日常→打破日常的契机）、第二幕（冲突升级/障碍与成长）、第三幕（高潮与结局）。每幕列出关键情节点。用 Markdown 格式输出。',
  },
  {
    id: 5,
    title: '章节大纲 · 目录',
    desc: '按主线切分卷 / 章，每章一句话。完成后可切到「写作」模式。',
    status: 'pending',
    settingType: 'outline',
    prompt: '请根据前面的梗概、世界观、人物和主线，切分章节大纲。按「卷→章」结构，每章给一句话标题和一句概要。用 Markdown 格式输出。',
  },
])

function statusLabel(s: string) {
  return s === 'done' ? '已完成' : s === 'doing' ? '进行中' : '未开始'
}

async function startStep(id: number) {
  const step = steps.value.find(s => s.id === id)
  if (!step || step.status === 'doing') return

  step.status = 'doing'
  step.error = undefined
  step.preview = undefined

  try {
    // 收集已完成步骤的结果作为上下文
    const contextParts: string[] = [`项目名：${props.projectName || '未命名'}`]
    steps.value.filter(s => s.status === 'done' && s.preview).forEach(s => {
      contextParts.push(`【${s.title}】\n${s.preview}`)
    })

    const res = await aiApi.generate({
      project_id: String(props.projectId),
      mode: 'plan',
      content: `${step.prompt}\n\n已有构思参考：\n${contextParts.join('\n\n')}`,
      stream: false,
      include_settings: true,
    })

    const data = res.data?.data || res.data
    const result = data?.content || ''
    if (!result) throw new Error('AI 返回为空')

    step.preview = result
    step.status = 'done'
    step.editing = false
  } catch (e: any) {
    step.status = 'pending'
    step.error = e.response?.data?.error || e.message || 'AI 生成失败，请重试'
  }
}

// 查找项目里对应 type 的设定文档，把结果写入
async function writeToSetting(step: PlanStep) {
  step.writing = true
  try {
    const res = await documentApi.list(String(props.projectId))
    const docs = res.data?.data || res.data || []
    const folder = docs.find((d: any) => d.type === step.settingType && !d.parent_id)

    if (!folder) {
      step.writing = false
      alert('未找到对应设定文件夹')
      return
    }

    const content = `# ${step.title}\n\n${step.preview}`
    // ponytail: 同名子文档存在则更新，否则新建
    const existing = docs.find((d: any) => d.parent_id === folder.id && d.title === step.title)
    if (existing) {
      await documentApi.update(existing.id, { content })
      step.writing = false
      emit('jump-to-doc', existing.id)
    } else {
      const createRes = await documentApi.create({
        project_id: Number(props.projectId),
        parent_id: folder.id,
        type: step.settingType,
        title: step.title,
        content,
      })
      step.writing = false
      const newId = createRes.data?.data?.id || createRes.data?.id
      if (newId) emit('jump-to-doc', newId)
    }
  } catch (e: any) {
    step.writing = false
    alert(e.response?.data?.error || '写入失败，请重试')
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
  margin-bottom: 24px;
  font-size: 13px;
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.step-card {
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  padding: 18px 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
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
.step-status.pending { background: rgba(0, 0, 0, 0.05); color: var(--text-secondary); }
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
  background: rgba(0, 0, 0, 0.02);
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
</style>
