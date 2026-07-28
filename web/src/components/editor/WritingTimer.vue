<template>
  <div class="writing-timer" :class="state">
    <!-- Summary view (shown after stop, until dismissed) -->
    <template v-if="state === 'summary'">
      <div class="timer-summary">
        <span class="summary-time">{{ formatTime(summaryMs) }}</span>
        <span class="summary-sep">·</span>
        <span class="summary-words" :class="{ negative: summaryWords < 0 }">
          {{ summaryWords >= 0 ? '+' : '' }}{{ summaryWords }} 字
        </span>
        <button class="timer-btn dismiss" @click="reset" title="关闭">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
        </button>
      </div>
    </template>

    <!-- Active timer -->
    <template v-else>
      <span v-if="state === 'running'" class="pulse-dot"></span>
      <span class="timer-display">{{ displayTime }}</span>
      <span v-if="state !== 'idle'" class="words-delta" :class="{ negative: wordsDelta < 0 }">
        {{ wordsDelta >= 0 ? '+' : '' }}{{ wordsDelta }}
      </span>

      <div class="timer-controls">
        <!-- Play / Pause -->
        <button
          v-if="state === 'idle' || state === 'paused'"
          class="timer-btn play"
          @click="start"
          :title="state === 'paused' ? '继续' : '开始计时'"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 4 20 12 6 20"/></svg>
        </button>
        <button
          v-else
          class="timer-btn pause"
          @click="pause"
          title="暂停"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
        </button>

        <!-- Stop (only when running or paused) -->
        <button
          v-if="state === 'running' || state === 'paused'"
          class="timer-btn stop"
          @click="stop"
          title="结束并查看总结"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><rect x="5" y="5" width="14" height="14" rx="2"/></svg>
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, shallowRef } from 'vue'

const props = defineProps<{
  currentWordCount: number
}>()

type TimerState = 'idle' | 'running' | 'paused' | 'summary'

const state = ref<TimerState>('idle')
const intervalRef = shallowRef<ReturnType<typeof setInterval> | null>(null)

// Accumulated elapsed ms across pause/resume cycles
const accumulatedMs = ref(0)
const sessionStart = shallowRef<number>(0) // timestamp when current run-segment started

const startWordCount = ref(0)
const lastWordCount = ref(0) // tracks live word count for delta display

// Summary data
const summaryMs = ref(0)
const summaryWords = ref(0)

const now = ref(Date.now())

const elapsedMs = computed(() => {
  if (state.value === 'running') return accumulatedMs.value + (now.value - sessionStart.value)
  return accumulatedMs.value
})

const displayTime = computed(() => formatTime(elapsedMs.value))

const wordsDelta = computed(() => lastWordCount.value - startWordCount.value)

// ponytail: tick updates `now` so the display recomputes; 1s granularity is fine for MM:SS
function tick() {
  now.value = Date.now()
}

function start() {
  if (state.value === 'idle') {
    accumulatedMs.value = 0
    startWordCount.value = props.currentWordCount
    lastWordCount.value = props.currentWordCount
  }
  // paused → resume
  sessionStart.value = Date.now()
  state.value = 'running'
  if (intervalRef.value) clearInterval(intervalRef.value)
  intervalRef.value = setInterval(tick, 1000)
}

function pause() {
  if (state.value !== 'running') return
  accumulatedMs.value += Date.now() - sessionStart.value
  sessionStart.value = 0
  state.value = 'paused'
  if (intervalRef.value) { clearInterval(intervalRef.value); intervalRef.value = null }
}

function stop() {
  if (state.value === 'running') {
    accumulatedMs.value += Date.now() - sessionStart.value
    sessionStart.value = 0
  }
  if (intervalRef.value) { clearInterval(intervalRef.value); intervalRef.value = null }
  summaryMs.value = accumulatedMs.value
  summaryWords.value = props.currentWordCount - startWordCount.value
  state.value = 'summary'
}

function reset() {
  state.value = 'idle'
  accumulatedMs.value = 0
  startWordCount.value = 0
  lastWordCount.value = 0
  summaryMs.value = 0
  summaryWords.value = 0
}

// Live word count tracking while running
watch(() => props.currentWordCount, (v) => {
  if (state.value === 'running' || state.value === 'paused') {
    lastWordCount.value = v
  }
})

function formatTime(ms: number): string {
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}

onBeforeUnmount(() => {
  if (intervalRef.value) clearInterval(intervalRef.value)
})
</script>

<style scoped>
.writing-timer {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: 7px;
  background: var(--segment-bg, rgba(0,0,0,0.05));
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: var(--text-secondary, #86868b);
  transition: all var(--transition-fast, 0.15s ease);
  white-space: nowrap;
}

.writing-timer.running {
  background: var(--accent-light, rgba(0,122,255,0.08));
  color: var(--accent, #007aff);
  box-shadow: 0 0 0 1px var(--accent-light, rgba(0,122,255,0.15));
}
.writing-timer.paused {
  color: var(--text-secondary, #86868b);
  opacity: 0.8;
}
.writing-timer.summary {
  background: var(--accent-light, rgba(0,122,255,0.08));
  color: var(--accent, #007aff);
}

.timer-display {
  font-weight: 600;
  letter-spacing: 0.02em;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent, #007aff);
  animation: timer-pulse 1.5s ease-in-out infinite;
}
@keyframes timer-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.8); }
}

.words-delta {
  font-size: 11px;
  opacity: 0.8;
  padding-left: 2px;
  border-left: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  padding-left: 6px;
}
.words-delta.negative { color: var(--text-tertiary, #aeaeb2); }

.timer-controls {
  display: inline-flex;
  gap: 2px;
  margin-left: 2px;
}

.timer-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 5px;
  background: transparent;
  cursor: pointer;
  color: inherit;
  padding: 0;
  transition: background var(--transition-fast, 0.15s ease);
}
.timer-btn:hover {
  background: var(--hover-bg, rgba(0,0,0,0.06));
}
.writing-timer.running .timer-btn:hover {
  background: rgba(0,122,255,0.12);
}

.timer-btn.play { color: var(--accent, #007aff); }

/* Summary */
.timer-summary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.summary-time {
  font-weight: 600;
}
.summary-sep {
  opacity: 0.5;
}
.summary-words {
  font-size: 11px;
}
.summary-words.negative { color: var(--text-tertiary, #aeaeb2); }
.timer-btn.dismiss {
  width: 16px;
  height: 16px;
  margin-left: 2px;
  opacity: 0.6;
}
.timer-btn.dismiss:hover { opacity: 1; }
</style>
