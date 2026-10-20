<template>
  <div class="stats-panel">
    <div class="stats-header">
      <span>📊 写作统计</span>
    </div>
    <div v-if="loading" class="stats-loading">加载中...</div>
    <div v-else-if="!stats" class="stats-empty">暂无数据</div>
    <template v-else>
      <!-- 核心数字 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-num today">{{ formatNum(stats.today_words) }}</div>
          <div class="stat-label">今日</div>
        </div>
        <div class="stat-card">
          <div class="stat-num week">{{ formatNum(stats.week_words) }}</div>
          <div class="stat-label">本周</div>
        </div>
        <div class="stat-card">
          <div class="stat-num month">{{ formatNum(stats.month_words) }}</div>
          <div class="stat-label">本月</div>
        </div>
        <div class="stat-card">
          <div class="stat-num streak">{{ stats.streak_days }}</div>
          <div class="stat-label">连续🔥天</div>
        </div>
      </div>

      <!-- 全书总字数 -->
      <div class="stat-total-row">
        <span>全书总字数</span>
        <strong>{{ formatNum(stats.total_words) }}</strong>
      </div>

      <!-- 14天柱状图 -->
      <div class="stat-chart" v-if="stats.daily?.length">
        <div class="chart-bars">
          <div
            v-for="d in stats.daily"
            :key="d.date"
            class="bar-col"
            :title="`${d.date}: ${d.words}字`"
          >
            <div
              class="bar"
              :style="{ height: barHeight(d.words) + '%' }"
              :class="{ today: isToday(d.date) }"
            ></div>
            <span class="bar-label">{{ d.date.slice(5) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { projectApi } from '../../core/api'

interface StatsData {
  today_words: number
  week_words: number
  month_words: number
  total_words: number
  streak_days: number
  daily: { date: string; words: number }[]
}

const props = defineProps<{ projectId: number }>()

const loading = ref(false)
const stats = ref<StatsData | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await projectApi.stats(props.projectId)
    stats.value = res.data?.data ?? null
  } catch {
    stats.value = null
  } finally {
    loading.value = false
  }
}

// ponytail: reuse on mount + watch projectId
onMounted(load)
watch(() => props.projectId, load)

const maxDaily = () => Math.max(1, ...(stats.value?.daily?.map(d => d.words) || [1]))

function barHeight(words: number) {
  return Math.max(words > 0 ? 8 : 2, (words / maxDaily()) * 100)
}

function isToday(date: string) {
  return date === new Date().toISOString().slice(0, 10)
}

function formatNum(n: number) {
  return n >= 10000 ? `${(n / 10000).toFixed(1)}万` : n.toLocaleString()
}
</script>

<style scoped>
.stats-panel {
  width: 360px;
  max-height: 480px;
  overflow-y: auto;
  background: var(--bg-primary, #fff);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.12);
  padding: 16px;
}
.stats-header {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
}
.stats-loading, .stats-empty {
  text-align: center;
  color: var(--text-tertiary);
  padding: 24px 0;
  font-size: 13px;
}
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}
.stat-card {
  text-align: center;
  background: var(--bg-secondary, #f5f5f7);
  border-radius: 8px;
  padding: 10px 4px;
}
.stat-num {
  font-size: 20px;
  font-weight: 700;
  font-family: var(--font-serif);
  color: var(--text-primary);
}
.stat-num.today { color: var(--accent, #007AFF); }
.stat-num.streak { color: #ff6b35; }
.stat-label {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 2px;
}
.stat-total-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  font-size: 13px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border, #e5e5e7);
  border-bottom: 1px solid var(--border, #e5e5e7);
  margin-bottom: 12px;
}
.stat-total-row strong {
  font-size: 16px;
  color: var(--text-primary);
}
.stat-chart {
  margin-top: 4px;
}
.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 80px;
}
.bar-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  height: 100%;
}
.bar {
  width: 100%;
  max-width: 16px;
  background: rgba(0,122,255,0.2);
  border-radius: 3px 3px 0 0;
  min-height: 2px;
  transition: height 0.3s ease;
}
.bar.today {
  background: var(--accent, #007AFF);
}
.bar-label {
  font-size: 8px;
  color: var(--text-tertiary);
  margin-top: 2px;
  transform: rotate(-45deg);
  transform-origin: center;
  white-space: nowrap;
}
</style>
