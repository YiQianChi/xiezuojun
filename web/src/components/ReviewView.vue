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
        <button v-if="history.length > 0" class="clear-btn" @click="clearHistory" title="清空历史报告">🗑</button>
      </div>

      <div v-if="history.length > 0" class="review-history">
        <select @change="selectReport(Number(($event.target as HTMLSelectElement).value))" :value="activeReportTs ?? ''">
          <option v-for="r in history" :key="r.timestamp" :value="r.timestamp">
            {{ new Date(r.timestamp).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) }} · {{ r.issues.length }}个问题
          </option>
        </select>
      </div>

      <div v-if="issues.length === 0 && !running && !error" class="empty-state">
        <div class="empty-icon">🔍</div>
        <div class="empty-title">还没有体检报告</div>
        <div class="empty-desc">点上面「开始体检」，AI 会当读者/责编，替你查一致性、伏笔、节奏。</div>
      </div>

      <div v-if="error" class="empty-state">
        <div class="empty-icon">⚠️</div>
        <div class="empty-title">检查失败</div>
        <div class="empty-desc">{{ error }}</div>
      </div>

      <div v-if="running" class="empty-state">
        <div class="empty-icon">⏳</div>
        <div class="empty-title">AI 正在审阅你的小说...</div>
        <div class="empty-desc">逐章检查中，请稍候</div>
      </div>

      <template v-if="!running">
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

        <div v-if="activeIssue.suggestion" class="ai-note">
          <strong>AI 建议：</strong>
          <div class="ai-note-content">{{ activeIssue.suggestion }}</div>
        </div>

        <div class="action-row">
          <button class="btn" @click="activeIssue && emit('jump-to-source', activeIssue)">跳到原文</button>
          <button class="btn primary" @click="activeIssue && emit('ai-fix', activeIssue)">让 AI 帮我改</button>
          <button class="btn" @click="ignoreAndNext">忽略</button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { aiApi, documentApi } from '../core/api'

const emit = defineEmits<{
  (e: 'jump-to-source', issue: ReviewIssue): void
  (e: 'ai-fix', issue: ReviewIssue): void
}>()

const props = defineProps<{
  projectId: number
}>()

interface ReviewIssue {
  id: string
  category: string
  title: string
  desc: string
  severity: 'high' | 'med' | 'low'
  locus: string
  chapter_title?: string
  suggestion?: string
}

interface ReviewReport {
  timestamp: number
  scope: string
  issues: ReviewIssue[]
}

const scope = ref<'book' | 'volume' | 'chapter'>('book')
const running = ref(false)
const error = ref('')
const ignoredIds = ref(new Set<string>())
const activeId = ref<string | null>(null)
const issues = ref<ReviewIssue[]>([])
const history = ref<ReviewReport[]>([])
const activeReportTs = ref<number | null>(null)

const historyKey = computed(() => `review-${props.projectId}`)

function loadHistory() {
  try {
    const raw = localStorage.getItem(historyKey.value)
    if (!raw) return
    const arr = JSON.parse(raw) as ReviewReport[]
    history.value = Array.isArray(arr) ? arr.sort((a, b) => b.timestamp - a.timestamp) : []
  } catch { /* ponytail: corrupt localStorage, ignore */ }
}

function saveHistory() {
  try {
    localStorage.setItem(historyKey.value, JSON.stringify(history.value))
  } catch { /* ponytail: quota exceeded, ignore */ }
}

function selectReport(ts: number) {
  const report = history.value.find(r => r.timestamp === ts)
  if (!report) return
  activeReportTs.value = ts
  issues.value = report.issues
  ignoredIds.value = new Set()
  activeId.value = report.issues[0]?.id ?? null
}

function clearHistory() {
  history.value = []
  activeReportTs.value = null
  issues.value = []
  activeId.value = null
  localStorage.removeItem(historyKey.value)
}

const categories = ['人设一致性', '情节漏洞', '伏笔追踪', '节奏']

function issuesByCategory(cat: string) {
  return issues.value.filter(i => i.category === cat && !ignoredIds.value.has(i.id))
}

const visibleIssues = computed(() => issues.value.filter(i => !ignoredIds.value.has(i.id)))
const activeIssue = computed(() => visibleIssues.value.find(i => i.id === activeId.value) || null)

function ignoreAndNext() {
  if (!activeIssue.value) return
  ignoredIds.value.add(activeIssue.value.id)
  const next = visibleIssues.value.find(i => i.id !== activeIssue.value!.id)
  activeId.value = next?.id ?? null
}

function severityLabel(s: string) {
  return s === 'high' ? '高' : s === 'med' ? '中' : '低'
}

// 收集项目正文章节文本，传给 AI 审阅
async function collectChapterText(): Promise<string> {
  const res = await documentApi.list(String(props.projectId))
  const docs = res.data?.data || res.data || []
  const chapters = docs.filter((d: any) => d.type === 'chapter' && d.search_text)

  if (scope.value === 'chapter') {
    const ch = chapters[0]
    return ch ? `【${ch.title}】\n${ch.search_text?.slice(0, 12000) || ''}` : ''
  }

  // book / volume: 拼接所有章节正文
  const parts: string[] = []
  let total = 0
  for (const ch of chapters) {
    if (total > 24000) {
      parts.push('（剩余章节因篇幅限制省略...）')
      break
    }
    const text = ch.search_text || ''
    parts.push(`【${ch.title}】\n${text}`)
    total += text.length
  }
  return parts.join('\n\n')
}

async function runReview() {
  running.value = true
  error.value = ''
  issues.value = []

  try {
    const chapterText = await collectChapterText()
    if (!chapterText.trim()) {
      error.value = '项目还没有正文内容，无法审阅'
      running.value = false
      return
    }

    const res = await aiApi.generate({
      project_id: String(props.projectId),
      mode: 'review',
      content: `请审阅以下小说正文，检查人设一致性、情节漏洞、伏笔追踪和节奏问题。\n每个 issue 必须包含 chapter_title 字段，值为该问题所在章节的精确标题。\n\n${chapterText}`,
      stream: false,
    })

    const data = res.data?.data || res.data
    const raw = data?.content || ''

    // 解析 JSON —— GLM 可能带 ```json 包裹
    let jsonStr = raw.trim()
    const fenceMatch = jsonStr.match(/```(?:json)?\s*([\s\S]*?)```/)
    if (fenceMatch) jsonStr = fenceMatch[1].trim()

    // 找第一个 { 到最后一个 }
    const firstBrace = jsonStr.indexOf('{')
    const lastBrace = jsonStr.lastIndexOf('}')
    if (firstBrace >= 0 && lastBrace > firstBrace) {
      jsonStr = jsonStr.slice(firstBrace, lastBrace + 1)
    }

    const parsed = JSON.parse(jsonStr)
    const rawIssues = parsed.issues || []

    issues.value = rawIssues.map((item: any, idx: number) => ({
      id: `issue-${idx}`,
      category: item.category || '其他',
      title: item.title || '未命名问题',
      desc: item.desc || item.description || '',
      severity: ['high', 'med', 'low'].includes(item.severity) ? item.severity : 'med',
      locus: item.locus || item.location || '未知位置',
      chapter_title: item.chapter_title || undefined,
      suggestion: item.suggestion || undefined,
    }))

    if (issues.value.length > 0) {
      activeId.value = issues.value[0].id
    }

    // save report to history
    const report: ReviewReport = {
      timestamp: Date.now(),
      scope: scope.value,
      issues: [...issues.value],
    }
    history.value.unshift(report)
    activeReportTs.value = report.timestamp
    saveHistory()
  } catch (e: any) {
    error.value = e.response?.data?.error || e.message || 'AI 审阅失败，请重试'
  } finally {
    running.value = false
  }
}

onMounted(() => {
  loadHistory()
  if (history.value.length > 0) {
    selectReport(history.value[0].timestamp)
  }
})
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
.review-scope .clear-btn {
  background: none;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 13px;
  cursor: pointer;
  color: var(--text-secondary);
}
.review-history {
  padding: 8px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}
.review-history select {
  width: 100%;
  background: var(--bg-primary, #f5f5f7);
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--text-primary);
}

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
