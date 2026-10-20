<template>
  <Teleport to="body">
    <!-- 收起态：小圆钮 -->
    <button v-if="!visible && state !== 'idle'" class="buddy-fab" @click="visible = true" title="结伴写作">
      <span class="fab-dot"></span>
      <span class="fab-time">{{ displayTime }}</span>
    </button>

    <!-- 悬浮窗 -->
    <div v-if="visible" class="buddy-window" :style="winStyle" @mousedown="onDragStart">
      <div class="bw-head">
        <span class="bw-title">
          <span class="pulse-dot" v-if="state === 'running'"></span>
          {{ displayTime }}
          <span class="bw-delta" v-if="state !== 'idle'" :class="{ negative: wordsDelta < 0 }">{{ wordsDelta >= 0 ? '+' : '' }}{{ wordsDelta }}</span>
        </span>
        <span class="bw-actions">
          <button v-if="state === 'idle'" class="bw-btn play" @click="start" title="开始计时">▶</button>
          <button v-else-if="state === 'running'" class="bw-btn" @click="pause" title="暂停">⏸</button>
          <button v-else class="bw-btn play" @click="start" title="继续">▶</button>
          <button v-if="state !== 'idle'" class="bw-btn" @click="stop" title="结束">⏹</button>
          <button class="bw-btn close" @click="visible = false" title="收起">—</button>
        </span>
      </div>

      <div class="bw-body">
        <!-- 未开始 -->
        <div v-if="state === 'idle'" class="bw-idle">
          <p class="bw-hint">开始计时后可以找一个<br>同样在写的搭子互相陪伴</p>
        </div>

        <!-- 计时中：搭子区 -->
        <div v-else-if="state === 'running' || state === 'paused'" class="bw-buddy">
          <template v-if="peer">
            <div class="bw-peer">
              <div class="bw-avatar">{{ (peer.nickname || '友')[0] }}</div>
              <div class="bw-peer-info">
                <div class="bw-peer-name">{{ peer.nickname || '写作搭子' }}</div>
                <div class="bw-peer-stat">
                  <span class="bw-peer-time">{{ peerTime }}</span>
                  <span class="bw-peer-words" v-if="peer.words_delta > 0">+{{ peer.words_delta }} 字</span>
                </div>
              </div>
              <button class="bw-cheer" @click="cheer" :class="{ sent: cheerJustSent }">👏 加油</button>
            </div>
            <div v-if="peerCheerFlash" class="bw-cheer-flash">💛 搭子给你加了油！</div>
            <div class="bw-online">当前 {{ online }} 人在写</div>
          </template>
          <template v-else-if="waiting">
            <div class="bw-waiting">
              <span class="spinner"></span> 正在寻找搭子…
              <button class="bw-btn cancel" @click="leaveMatch">取消</button>
            </div>
          </template>
          <template v-else>
            <button class="bw-match-btn" @click="match">🔍 找个搭子</button>
            <div class="bw-online">当前 {{ online }} 人在写</div>
          </template>
        </div>

        <!-- 总结 -->
        <div v-else-if="state === 'summary'" class="bw-summary">
          <span class="sum-time">{{ formatTime(summaryMs) }}</span>
          <span class="sum-sep">·</span>
          <span class="sum-words" :class="{ negative: summaryWords < 0 }">{{ summaryWords >= 0 ? '+' : '' }}{{ summaryWords }} 字</span>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { buddyApi } from '../../core/api'

const props = defineProps<{ currentWordCount: number }>()

type TimerState = 'idle' | 'running' | 'paused' | 'summary'
const STORAGE_KEY = 'xzj-buddy-timer'
const MAX_RESUME_GAP_MS = 60 * 60 * 1000

const visible = ref(true)
const state = ref<TimerState>('idle')
const intervalRef = shallowRef<ReturnType<typeof setInterval> | null>(null)
const beatRef = shallowRef<ReturnType<typeof setInterval> | null>(null)
const accumulatedMs = ref(0)
const sessionStart = shallowRef(0)
const startWordCount = ref(0)
const lastWordCount = ref(0)
const summaryMs = ref(0)
const summaryWords = ref(0)
const now = ref(Date.now())

// 搭子状态
interface PeerSession { user_id: string; nickname: string; started_at: string; words_delta: number }
const peer = ref<PeerSession | null>(null)
const waiting = ref(false)
const online = ref(0)
const peerCheerFlash = ref(false)
const cheerJustSent = ref(false)

// 窗口拖动
const winPos = ref({ x: 0, y: 0 })
const winStyle = computed(() => {
  if (winPos.value.x === 0 && winPos.value.y === 0) return {}
  return { right: 'auto', bottom: 'auto', left: winPos.value.x + 'px', top: winPos.value.y + 'px' }
})
let dragging = false, dragOffX = 0, dragOffY = 0
function onDragStart(e: MouseEvent) {
  if ((e.target as HTMLElement).closest('button')) return
  dragging = true
  const el = document.querySelector('.buddy-window') as HTMLElement
  const r = el.getBoundingClientRect()
  dragOffX = e.clientX - r.left
  dragOffY = e.clientY - r.top
  e.preventDefault()
}
function onDragMove(e: MouseEvent) {
  if (!dragging) return
  winPos.value = { x: Math.max(0, e.clientX - dragOffX), y: Math.max(0, e.clientY - dragOffY) }
}
function onDragEnd() { dragging = false }

const elapsedMs = computed(() => {
  if (state.value === 'running') return accumulatedMs.value + (now.value - sessionStart.value)
  return accumulatedMs.value
})
const displayTime = computed(() => formatTime(elapsedMs.value))
const wordsDelta = computed(() => lastWordCount.value - startWordCount.value)
const peerTime = computed(() => {
  if (!peer.value) return ''
  const ms = Date.now() - new Date(peer.value.started_at).getTime()
  return formatTime(Math.max(0, ms))
})

function tick() {
  now.value = Date.now()
  saveSnapshot()
}

// ---- 后端交互 ----
async function heartbeat() {
  if (state.value !== 'running' && state.value !== 'paused') return
  try {
    const res = await buddyApi.heartbeat({
      project_id: Number(localStorage.getItem('xzj-active-project') || 0),
      words_delta: wordsDelta.value,
    })
    applyState(res.data?.data)
  } catch { /* 网络断就下一拍再试 */ }
}

function applyState(d: any) {
  if (!d) return
  const s = d.state || {}
  online.value = s.online || 0
  waiting.value = !!s.waiting
  peer.value = s.peer || null
  if (s.peer_cheer > 0) {
    peerCheerFlash.value = true
    setTimeout(() => { peerCheerFlash.value = false }, 4000)
  }
}

async function match() {
  try {
    const res = await buddyApi.match()
    if (res.data?.data?.matched) {
      waiting.value = false
      peer.value = res.data.data.peer
    } else {
      waiting.value = true
    }
  } catch { }
}
async function leaveMatch() {
  try { await buddyApi.leaveMatch() } catch { }
  waiting.value = false
  peer.value = null
}
async function cheer() {
  try {
    await buddyApi.cheer()
    cheerJustSent.value = true
    setTimeout(() => { cheerJustSent.value = false }, 2000)
  } catch { }
}

// ---- 计时器逻辑（同 WritingTimer，含持久化） ----
function saveSnapshot() {
  const snap = {
    state: state.value, accumulatedMs: accumulatedMs.value, sessionStart: sessionStart.value,
    startWordCount: startWordCount.value, lastWordCount: lastWordCount.value,
    summaryMs: summaryMs.value, summaryWords: summaryWords.value,
    visible: visible.value, win: winPos.value,
  }
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(snap)) } catch { }
}
function restoreSnapshot() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const s = JSON.parse(raw)
    visible.value = s.visible !== false
    if (s.win && (s.win.x || s.win.y)) winPos.value = s.win
    if (s.state === 'running') {
      const gap = Date.now() - s.sessionStart
      if (gap > MAX_RESUME_GAP_MS) {
        accumulatedMs.value = s.accumulatedMs
        state.value = 'paused'
      } else {
        accumulatedMs.value = s.accumulatedMs + gap
        sessionStart.value = Date.now()
        state.value = 'running'
        now.value = Date.now()
        if (intervalRef.value) clearInterval(intervalRef.value)
        intervalRef.value = setInterval(tick, 1000)
      }
    } else if (s.state === 'paused' || s.state === 'summary') {
      state.value = s.state
      accumulatedMs.value = s.accumulatedMs
    } else { return }
    startWordCount.value = s.startWordCount
    lastWordCount.value = s.lastWordCount
    summaryMs.value = s.summaryMs
    summaryWords.value = s.summaryWords
  } catch { }
}
function clearSnapshot() { try { localStorage.removeItem(STORAGE_KEY) } catch { } }

function start() {
  if (state.value === 'idle') {
    accumulatedMs.value = 0
    startWordCount.value = props.currentWordCount
    lastWordCount.value = props.currentWordCount
  }
  sessionStart.value = Date.now()
  state.value = 'running'
  if (intervalRef.value) clearInterval(intervalRef.value)
  intervalRef.value = setInterval(tick, 1000)
  if (!beatRef.value) beatRef.value = setInterval(heartbeat, 15000)
  heartbeat()
  saveSnapshot()
}
function pause() {
  if (state.value !== 'running') return
  accumulatedMs.value += Date.now() - sessionStart.value
  sessionStart.value = 0
  state.value = 'paused'
  if (intervalRef.value) { clearInterval(intervalRef.value); intervalRef.value = null }
  saveSnapshot()
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
  leaveMatch()
  if (beatRef.value) { clearInterval(beatRef.value); beatRef.value = null }
  buddyApi.stop().catch(() => { })
  saveSnapshot()
}
function reset() {
  state.value = 'idle'
  accumulatedMs.value = 0
  startWordCount.value = 0
  lastWordCount.value = 0
  summaryMs.value = 0
  summaryWords.value = 0
  clearSnapshot()
}
defineExpose({ reset })

function formatTime(ms: number): string {
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}

onMounted(() => {
  restoreSnapshot()
  document.addEventListener('mousemove', onDragMove)
  document.addEventListener('mouseup', onDragEnd)
})
onBeforeUnmount(() => {
  saveSnapshot()
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
  if (intervalRef.value) clearInterval(intervalRef.value)
  if (beatRef.value) clearInterval(beatRef.value)
})
</script>

<style scoped>
.buddy-fab {
  position: fixed; right: 24px; bottom: 44px; z-index: 900;
  display: flex; align-items: center; gap: 6px;
  padding: 8px 12px; border-radius: 999px; border: none; cursor: pointer;
  background: var(--bg-secondary, #fff); color: var(--text-primary, #333);
  box-shadow: 0 4px 20px rgba(0,0,0,0.18), 0 0 0 1px rgba(0,0,0,0.06);
  font-size: 12px; font-variant-numeric: tabular-nums; font-weight: 600;
}
.buddy-fab .fab-dot { width: 7px; height: 7px; border-radius: 50%; background: #30d158; animation: bfab 1.5s infinite; }
@keyframes bfab { 50% { opacity: .35; } }

.buddy-window {
  position: fixed; right: 24px; bottom: 44px; z-index: 900;
  width: 230px; border-radius: 14px; overflow: hidden;
  background: var(--bg-secondary, #fff);
  box-shadow: 0 8px 32px rgba(0,0,0,0.2), 0 0 0 1px rgba(0,0,0,0.06);
  user-select: none;
}
.bw-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 10px 6px; cursor: grab;
  border-bottom: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
}
.bw-head:active { cursor: grabbing; }
.bw-title { display: flex; align-items: center; gap: 6px; font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary, #333); }
.pulse-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent, #007aff); animation: bfab 1.5s infinite; }
.bw-delta { font-size: 11px; font-weight: 500; color: #30d158; }
.bw-delta.negative { color: var(--text-tertiary, #999); }
.bw-actions { display: flex; gap: 2px; }
.bw-btn {
  width: 22px; height: 22px; border: none; border-radius: 6px; background: transparent;
  cursor: pointer; font-size: 11px; color: var(--text-secondary, #666);
  display: inline-flex; align-items: center; justify-content: center;
}
.bw-btn:hover { background: var(--bg-hover, rgba(0,0,0,0.06)); }
.bw-btn.play { color: var(--accent, #007aff); }

.bw-body { padding: 10px 12px 12px; }
.bw-idle .bw-hint { font-size: 11.5px; color: var(--text-tertiary, #999); line-height: 1.7; text-align: center; padding: 4px 0; }

.bw-peer { display: flex; align-items: center; gap: 8px; }
.bw-avatar {
  width: 32px; height: 32px; border-radius: 50%; flex-shrink: 0;
  background: rgba(0,122,255,0.12); color: var(--accent, #007aff);
  display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 700;
}
.bw-peer-info { flex: 1; min-width: 0; }
.bw-peer-name { font-size: 12px; font-weight: 600; color: var(--text-primary, #333); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bw-peer-stat { display: flex; gap: 6px; font-size: 10.5px; color: var(--text-tertiary, #999); font-variant-numeric: tabular-nums; }
.bw-peer-words { color: #30d158; }
.bw-cheer {
  border: none; border-radius: 8px; padding: 5px 8px; cursor: pointer; flex-shrink: 0;
  background: rgba(255,204,0,0.15); color: #b8860b; font-size: 11px; font-weight: 600;
  transition: transform 0.15s ease;
}
.bw-cheer:hover { transform: scale(1.06); }
.bw-cheer.sent { background: rgba(255,204,0,0.35); }

.bw-cheer-flash {
  margin-top: 8px; padding: 6px 8px; border-radius: 8px; text-align: center;
  background: rgba(255,204,0,0.12); color: #b8860b; font-size: 11.5px; font-weight: 600;
  animation: cheerin 0.4s ease;
}
@keyframes cheerin { from { transform: scale(0.8); opacity: 0; } }

.bw-online { margin-top: 8px; font-size: 10.5px; color: var(--text-tertiary, #999); text-align: center; }

.bw-match-btn {
  width: 100%; padding: 8px; border: none; border-radius: 9px; cursor: pointer;
  background: rgba(0,122,255,0.1); color: var(--accent, #007aff);
  font-size: 12.5px; font-weight: 600;
}
.bw-match-btn:hover { background: rgba(0,122,255,0.18); }

.bw-waiting { display: flex; align-items: center; gap: 8px; justify-content: center; font-size: 12px; color: var(--text-secondary, #666); padding: 6px 0; }
.spinner {
  width: 13px; height: 13px; border-radius: 50%;
  border: 2px solid var(--border, #ddd); border-top-color: var(--accent, #007aff);
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.bw-btn.cancel { width: auto; padding: 0 8px; font-size: 11px; }

.bw-summary { display: flex; align-items: center; justify-content: center; gap: 6px; padding: 4px 0; cursor: pointer; }
.bw-summary:hover::after { content: '✕'; font-size: 10px; color: var(--text-tertiary); margin-left: 4px; }
.sum-time { font-size: 15px; font-weight: 700; color: var(--accent, #007aff); font-variant-numeric: tabular-nums; }
.sum-sep { color: var(--text-tertiary, #999); }
.sum-words { font-size: 12px; color: #30d158; }
.sum-words.negative { color: var(--text-tertiary, #999); }

@media (prefers-color-scheme: dark) {
  .buddy-window, .buddy-fab { background: #2c2c2e; color: #e5e5ea; }
  .bw-title, .bw-peer-name { color: #e5e5ea; }
}
</style>
