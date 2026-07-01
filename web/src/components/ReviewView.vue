<template>
  <div class="review-view">
    <div class="review-list">
      <div class="review-scope">
        <span>范围：</span>
        <select v-model="scope">
          <option value="book">整本书</option>
          <option value="volume">当前卷</option>
          <option value="chapter">本章</option>
        </select>
        <button class="run-btn" @click="runReview" :disabled="running">
          {{ running ? '检查中...' : '开始体检' }}
        </button>
      </div>

      <div v-if="issues.length === 0 && !running" class="empty-state">
        <div class="empty-icon">🔍</div>
        <div class="empty-title">还没有体检报告</div>
        <div class="empty-desc">点上面「开始体检」，AI 会当读者/责编，替你查一致性、伏笔、节奏。</div>
      </div>

      <template v-else>
        <template v-for="cat in categories" :key="cat">
          <div v-if="issuesByCategory(cat).length > 0" class="review-cat">{{ cat }}</div>
          <div
            v-for="issue in issuesByCategory(cat)"
            :key="issue.id"
            class="review-item"
            :class="{ active: activeId === issue.id }"
            @click="activeId = issue.id"
          >
            <div class="title">
              {{ issue.title }}
              <span class="severity" :class="issue.severity">{{ severityLabel(issue.severity) }}</span>
            </div>
            <div class="desc">{{ issue.desc }}</div>
            <div class="locus">📄 {{ issue.locus }}</div>
          </div>
        </template>
      </template>
    </div>

    <div class="review-detail">
      <div v-if="!activeIssue" class="detail-placeholder">
        <div class="ph-icon">📋</div>
        <div>选一条问题查看详细分析</div>
      </div>
      <template v-else>
        <h3>{{ activeIssue.title }}</h3>
        <div class="meta">
          <span class="badge" :class="activeIssue.severity">{{ activeIssue.category }} · {{ severityLabel(activeIssue.severity) }}</span>
          <span>📄 {{ activeIssue.locus }}</span>
        </div>
        <div class="summary">{{ activeIssue.desc }}</div>

        <template v-for="(q, i) in activeIssue.quotes" :key="i">
          <div class="quote-label">{{ q.label }}</div>
          <div class="quote">{{ q.text }}</div>
        </template>

        <div v-if="activeIssue.suggestion" class="ai-note">
          <strong>AI 建议：</strong>
          <div v-html="activeIssue.suggestion"></div>
        </div>

        <div class="action-row">
          <button class="btn">跳到原文</button>
          <button class="btn primary">让 AI 帮我改</button>
          <button class="btn">忽略</button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{
  projectId: number
}>()

interface ReviewIssue {
  id: string
  category: string
  title: string
  desc: string
  severity: 'high' | 'med' | 'low'
  locus: string
  quotes?: { label: string; text: string }[]
  suggestion?: string
}

const scope = ref<'book' | 'volume' | 'chapter'>('book')
const running = ref(false)
const activeId = ref<string | null>(null)

// 一期先做静态骨架，AI 后接
const issues = ref<ReviewIssue[]>([])

const categories = ['人设一致性', '情节漏洞', '伏笔追踪', '节奏']

function issuesByCategory(cat: string) {
  return issues.value.filter(i => i.category === cat)
}

const activeIssue = computed(() => issues.value.find(i => i.id === activeId.value) || null)

function severityLabel(s: string) {
  return s === 'high' ? '高' : s === 'med' ? '中' : '低'
}

async function runReview() {
  running.value = true
  // TODO: 后端 AI 接入
  setTimeout(() => {
    running.value = false
    alert('AI 审核模式正在接线中，敬请期待～')
  }, 500)
}
</script>

<style scoped>
.review-view {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-width: 0;
}

.review-list {
  width: 380px;
  background: var(--bg-secondary, #ffffff);
  border-right: 1px solid var(--border, #e5e5ea);
  overflow-y: auto;
  flex-shrink: 0;
}

.review-scope {
  padding: 14px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-secondary, #ffffff);
  position: sticky;
  top: 0;
  z-index: 1;
}
.review-scope select {
  flex: 1;
  background: var(--bg-primary, #f5f5f7);
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--text-primary);
}
.review-scope .run-btn {
  background: #ff9500;
  color: white;
  border: none;
  padding: 5px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-weight: 500;
}
.review-scope .run-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.empty-state {
  padding: 40px 24px;
  text-align: center;
}
.empty-icon { font-size: 32px; margin-bottom: 8px; }
.empty-title {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
  margin-bottom: 4px;
}
.empty-desc {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.review-cat {
  padding: 10px 16px 6px;
  font-size: 11px;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.review-item {
  padding: 10px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  cursor: pointer;
}
.review-item:hover { background: rgba(0, 0, 0, 0.02); }
.review-item.active {
  background: rgba(255, 149, 0, 0.06);
  border-left: 3px solid #ff9500;
  padding-left: 13px;
}
.review-item .title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.severity {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
}
.severity.high { background: rgba(255, 59, 48, 0.12); color: #ff3b30; }
.severity.med { background: rgba(255, 149, 0, 0.15); color: #ff9500; }
.severity.low { background: rgba(0, 122, 255, 0.1); color: #007aff; }

.review-item .desc {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}
.review-item .locus {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

.review-detail {
  flex: 1;
  padding: 30px 40px;
  overflow-y: auto;
  min-width: 0;
}
.detail-placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: 13px;
  gap: 12px;
}
.ph-icon { font-size: 40px; opacity: 0.5; }

.review-detail h3 {
  font-family: var(--font-serif);
  margin: 0 0 12px 0;
  font-size: 20px;
  color: var(--text-primary);
}
.meta {
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 16px;
  display: flex;
  gap: 12px;
  align-items: center;
}
.meta .badge {
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(255, 149, 0, 0.12);
  color: #ff9500;
  font-weight: 500;
}
.meta .badge.high { background: rgba(255, 59, 48, 0.12); color: #ff3b30; }
.meta .badge.med  { background: rgba(255, 149, 0, 0.15); color: #ff9500; }
.meta .badge.low  { background: rgba(0, 122, 255, 0.1); color: #007aff; }

.summary { color: var(--text-secondary); font-size: 13px; line-height: 1.7; }

.quote-label {
  margin-top: 16px;
  font-size: 12px;
  color: var(--text-tertiary);
}
.quote {
  background: var(--bg-primary, #f5f5f7);
  border-left: 3px solid #ff9500;
  padding: 12px 16px;
  font-family: var(--font-serif);
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-primary);
  margin: 6px 0 0 0;
  border-radius: 0 8px 8px 0;
}

.ai-note {
  background: rgba(0, 122, 255, 0.04);
  border-radius: 8px;
  padding: 14px 16px;
  margin-top: 16px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-primary);
}
.ai-note strong { color: #007aff; }

.action-row {
  display: flex;
  gap: 10px;
  margin-top: 20px;
  flex-wrap: wrap;
}
.action-row .btn {
  padding: 6px 14px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #ffffff);
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
}
.action-row .btn.primary {
  background: #ff9500;
  color: white;
  border-color: #ff9500;
}
</style>
