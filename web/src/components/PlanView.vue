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
          <div v-if="step.preview" class="step-preview" :class="{ overflow: step.preview.length > 120 }">{{ step.preview }}</div>
          <div class="step-actions">
            <button v-if="step.status === 'done'">重新生成</button>
            <button v-if="step.status === 'done'">编辑</button>
            <button v-if="step.status === 'done'" class="primary">写入设定 Tab</button>
            <button v-if="step.status === 'doing'">暂停</button>
            <button v-if="step.status === 'doing'" class="primary">查看进度</button>
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

defineProps<{
  projectName: string
}>()

interface PlanStep {
  id: number
  title: string
  desc: string
  status: 'pending' | 'doing' | 'done'
  preview?: string
}

// 一期先做静态骨架，AI 后接
const steps = ref<PlanStep[]>([
  {
    id: 1,
    title: '故事梗概',
    desc: '用一段话讲清楚：什么人 / 在什么背景 / 遇到什么事 / 怎么改变。',
    status: 'pending',
  },
  {
    id: 2,
    title: '世界观 · 关键设定',
    desc: '时代 / 地点 / 规则 / 冲突源头。AI 会生成多张世界观卡。',
    status: 'pending',
  },
  {
    id: 3,
    title: '人物关系',
    desc: '主要人物 3-5 人，各自的身份、动机、秘密。AI 会同步产出人物关系图。',
    status: 'pending',
  },
  {
    id: 4,
    title: '故事主线 · 三幕结构',
    desc: '开端 / 冲突升级 / 高潮与结局。AI 会给多版供你挑。',
    status: 'pending',
  },
  {
    id: 5,
    title: '章节大纲 · 目录',
    desc: '按主线切分卷 / 章，每章一句话。这一步完成就可以切到「写作」模式开写了。',
    status: 'pending',
  },
])

function statusLabel(s: string) {
  return s === 'done' ? '已完成' : s === 'doing' ? '进行中' : '未开始'
}

function startStep(_id: number) {
  // TODO: 后端 AI 接入。此处仅占位。
  alert('AI 计划模式正在接线中，敬请期待～')
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

.step-preview {
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  padding: 12px 14px;
  font-family: var(--font-serif);
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.7;
  margin-bottom: 10px;
  max-height: 100px;
  overflow: hidden;
  position: relative;
  white-space: pre-wrap;
}
.step-preview.overflow::after {
  content: '';
  position: absolute;
  bottom: 0; left: 0; right: 0; height: 30px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.02));
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
.step-actions button.primary {
  background: #af52de;
  color: white;
  border-color: #af52de;
}
.step-actions button.primary:hover {
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
