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
          <span v-if="state !== 'idle'" class="me-chip" :class="{ negative: wordsDelta < 0, zero: wordsDelta === 0 }">✍ {{ wordsDelta >= 0 ? '+' : '' }}{{ wordsDelta }}</span>
        </span>
        <span class="bw-actions">
          <button v-if="state === 'idle'" class="bw-btn play" @click="start" title="开始计时">▶</button>
          <button v-else-if="state === 'running'" class="bw-btn" @click="pause" title="暂停">⏸</button>
          <button v-else class="bw-btn play" @click="start" :title="state === 'summary' ? '开始新一场' : '继续'">▶</button>
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
                <div class="bw-peer-name"><span class="who ta">TA</span>{{ peer.nickname || '写作搭子' }}</div>
                <div class="bw-peer-stat">
                  <span class="bw-peer-time">{{ peerTime }}<span v-if="peerPaused" class="peer-paused-tag">⏸ 已暂停</span></span>
                  <span class="bw-peer-words" v-if="peer.words_delta > 0">+{{ peer.words_delta }} 字</span>
                  <span class="bw-peer-words revise" v-else-if="peer.words_delta < 0">{{ peer.words_delta }} 字 · 改稿中</span>
                </div>
              </div>
              <button class="bw-cheer" @click="cheer" :class="{ sent: cheerJustSent }">👏 加油</button>
            </div>
            <div v-if="peerCheerFlash" class="bw-cheer-flash">💛 搭子给你加了油！</div>
            <div v-if="peerMilestoneFlash" class="bw-ms-flash peer">✦ {{ peerMilestoneFlash }}</div>
            <div v-if="peerNudgeFlash" class="bw-nudge-flash">{{ peerNudgeFlash }}</div>
            <div class="bw-nudge-row">
              <button
                v-for="n in NUDGES" :key="n.id"
                class="bw-nudge-chip" :class="{ cd: nudgeCdLeft(n.id) > 0 }"
                @click="sendNudge(n)"
              >{{ n.icon }} {{ nudgeCdLeft(n.id) > 0 ? nudgeCdLeft(n.id) + 's' : n.label }}</button>
            </div>
          </template>
          <template v-else-if="waiting">
            <div class="bw-waiting">
              <span class="spinner"></span> 正在寻找搭子…
              <button class="bw-btn cancel" @click="leaveMatch">取消</button>
            </div>
          </template>
          <template v-else-if="rematchCountdown > 0">
            <div class="bw-waiting">
              <span>搭子已离开，{{ rematchCountdown }}s 后自动再找</span>
              <button class="bw-btn cancel" @click="onCancelRematchClick">不找了</button>
            </div>
          </template>
          <template v-else>
            <button class="bw-match-btn" @click="match">🔍 找个搭子</button>
          </template>
        </div>

        <!-- 总结：点击收工，回到未开始 -->
        <div v-else-if="state === 'summary'" class="bw-summary" @click="reset" title="点击收工，重新开始">
          <span class="sum-time">{{ formatTime(summaryMs) }}</span>
          <span class="sum-sep">·</span>
          <span class="sum-words" :class="{ negative: summaryWords < 0 }">{{ summaryWords >= 0 ? '+' : '' }}{{ summaryWords }} 字</span>
          <span class="sum-close">✕</span>
        </div>
        <div v-if="state === 'summary' && todayStat.words + todayStat.ms > 0" class="bw-today-line">
          今日累计 {{ formatTime(todayStat.ms) }} · {{ todayStat.words >= 0 ? '+' : '' }}{{ todayStat.words }} 字
        </div>

        <!-- 今日里程碑刻度条（今日累计口径：字数每 500 / 时长每 25 分钟） -->
        <div v-if="state !== 'summary' && state !== 'idle'" class="bw-ms-strip">
          <span
            v-for="i in 6" :key="i"
            class="ms-tick" :class="{ on: i <= milestoneTicks.done, now: i === milestoneTicks.done + 1 }"
          ></span>
          <span class="ms-next">今日 {{ milestoneTicks.done }} 档 · {{ milestoneTicks.nextLabel }}</span>
        </div>
      </div>
    </div>

    <!-- 自己的里程碑弹花（不打断输入，3 秒淡出） -->
    <div v-if="myMilestoneFlash" class="bw-ms-pop">
      <span class="ms-emoji">✦</span>
      <span class="ms-title">{{ myMilestoneFlash.title }}</span>
      <span class="ms-sub">{{ myMilestoneFlash.sub }}</span>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { buddyApi, userApi } from '../../core/api'

const props = defineProps<{ currentWordCount: number; activeDocId?: string | number | null }>()

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

// ---- 里程碑（今日累计口径）：今日字数每 500 字、今日码字时长每 25 分钟 ----
const MS_WORD_STEP = 500
const MS_TIME_STEP_MS = 25 * 60 * 1000
const myMilestoneFlash = ref<{ title: string; sub: string } | null>(null)
const peerMilestoneFlash = ref('')
let msWordLevel = 0 // 已达成的今日字数档（500 的倍数，来源服务端今日统计）
let msTimeLevel = 0 // 已达成的今日时长档（25 分钟的倍数）；字数档由服务端今日统计推
let msFlashTimer: ReturnType<typeof setTimeout> | null = null
let peerMsTimer: ReturnType<typeof setTimeout> | null = null

// 今日累计（服务端 Redis，净增口径；跨日自然清零）
const todayStat = ref<{ words: number; ms: number }>({ words: 0, ms: 0 })
function applyToday(d: any) {
  const t = d?.today
  if (t && typeof t.words === 'number' && typeof t.ms === 'number') {
    todayStat.value = { words: t.words, ms: t.ms }
  }
}
async function refreshToday() {
  try { applyToday((await buddyApi.today()).data?.data) } catch { }
}

const milestoneTicks = computed(() => {
  const wordLv = Math.max(0, Math.floor(todayStat.value.words / MS_WORD_STEP))
  const timeLv = Math.floor(Math.max(0, todayStat.value.ms) / MS_TIME_STEP_MS)
  const done = wordLv + timeLv
  const nextWord = (wordLv + 1) * MS_WORD_STEP
  const nextTimeMin = ((timeLv + 1) * MS_TIME_STEP_MS) / 60000
  const nextLabel = (nextWord - todayStat.value.words) <= (nextTimeMin * 60 - todayStat.value.ms / 1000) * 14
    ? `下一个 ${nextWord} 字`
    : `下一个 ${nextTimeMin} 分钟`
  return { total: done, done, nextLabel }
})

function checkMilestones() {
  if (state.value !== 'running') return
  const timeLv = Math.floor(Math.max(0, todayStat.value.ms) / MS_TIME_STEP_MS)
  const wordLv = Math.floor(Math.max(0, todayStat.value.words) / MS_WORD_STEP)
  if (wordLv > msWordLevel) {
    msWordLevel = wordLv
    fireMyMilestone(`今日破 ${wordLv * MS_WORD_STEP} 字！`, milestoneSub())
  } else if (timeLv > msTimeLevel) {
    msTimeLevel = timeLv
    fireMyMilestone(`今日 ${timeLv * 25} 分钟！`, milestoneSub())
  }
}
function milestoneSub(): string {
  const totalSec = Math.floor(elapsedMs.value / 1000)
  if (totalSec < 60) return ''
  const wpm = (wordsDelta.value / (totalSec / 60)).toFixed(1)
  return `${formatTime(elapsedMs.value)} · 平均 ${wpm} 字/分钟`
}
function fireMyMilestone(title: string, sub: string) {
  myMilestoneFlash.value = { title, sub }
  if (msFlashTimer) clearTimeout(msFlashTimer)
  msFlashTimer = setTimeout(() => { myMilestoneFlash.value = null }, 3000)
}
function flashPeerMilestone(text: string) {
  peerMilestoneFlash.value = text
  if (peerMsTimer) clearTimeout(peerMsTimer)
  peerMsTimer = setTimeout(() => { peerMilestoneFlash.value = '' }, 4000)
}

// 搭子状态
interface PeerSession {
  user_id: string; nickname: string; started_at: string
  words_delta: number; elapsed_sec?: number; timer_state?: string
}
const peer = ref<PeerSession | null>(null)
const waiting = ref(false)
const peerCheerFlash = ref(false)
const cheerJustSent = ref(false)
let clockOffset = 0 // 本机钟偏（server_time - Date.now()），校准对方时长
const peerElapsedBase = ref(0) // 最近一次快照时对方的累计码字毫秒
const peerPaused = computed(() => peer.value?.timer_state === 'paused')
// 对方时长：快照值 + running 时本地续滚（一秒一跳，暂停即停，钟偏已校准；依赖 now 每秒重算）
const peerTime = computed(() => {
  if (!peer.value) return ''
  const ms = peerElapsedBase.value + (peerPaused.value ? 0 : (now.value + clockOffset - peerSnapAt.value))
  return formatTime(Math.max(0, ms))
})
const peerSnapAt = ref(0) // 快照时刻（按服务器钟）

// 应用对方会话快照（心跳/SSE/matched 都走这里，保证口径一致）
function applyPeer(p: PeerSession | null) {
  if (!p) { peer.value = null; return }
  peer.value = p
  peerElapsedBase.value = (p.elapsed_sec || 0) * 1000
  peerSnapAt.value = Date.now() + clockOffset
}

// ---- 快捷互动（方案A）：预设 4 种，60s 冷却，只在对方窗内闪一条 4s 提示 ----
const NUDGES = [
  { id: 'water', icon: '💧', label: '喝杯水', recv: '让你喝杯水' },
  { id: 'rest', icon: '☕', label: '歇一会', recv: '让你歇一会' },
  { id: 'fire', icon: '🔥', label: '加把劲', recv: '给你加了把劲' },
  { id: 'together', icon: '🤝', label: '一起冲', recv: '邀你一起冲' },
] as const
const NUDGE_ICONS: Record<string, string> = { water: '💧', rest: '☕', fire: '🔥', together: '🤝' }
const NUDGE_RECV: Record<string, string> = { water: '让你喝杯水', rest: '让你歇一会', fire: '给你加了把劲', together: '邀你一起冲' }
const peerNudgeFlash = ref('')
const nudgeCooldown = ref<Record<string, number>>({}) // id → 冷却到期时间戳
const nudgeNow = ref(Date.now()) // 冷却倒计时的响应式时钟
let peerNudgeTimer: ReturnType<typeof setTimeout> | null = null
let nudgeTick: ReturnType<typeof setInterval> | null = null
function nudgeCdLeft(id: string): number {
  return Math.max(0, Math.ceil(((nudgeCooldown.value[id] || 0) - nudgeNow.value) / 1000))
}
function startNudgeTick() {
  if (nudgeTick) return
  nudgeTick = setInterval(() => {
    nudgeNow.value = Date.now()
    const active = Object.values(nudgeCooldown.value).some((t) => t > nudgeNow.value)
    if (!active && nudgeTick) { clearInterval(nudgeTick); nudgeTick = null }
  }, 1000)
}
async function sendNudge(n: typeof NUDGES[number]) {
  if (!peer.value || nudgeCdLeft(n.id) > 0) return
  nudgeCooldown.value = { ...nudgeCooldown.value, [n.id]: Date.now() + 60000 }
  startNudgeTick()
  try { await buddyApi.nudge(n.id) } catch { }
}

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

// ---- 字数增量按章节折叠：切换章节只重置基线，不把章节间字数差算成写作量 ----
// settledWords = 已结算的旧章节增量；curBaseline = 当前章节基线；
// 总增量 = settledWords + (当前章节字数 - 基线)，切章瞬间结算不跳变
const settledWords = ref(0)
const curBaseline = ref(0)
const wordsDelta = computed(() => settledWords.value + (lastWordCount.value - curBaseline.value))

// 字数跟随编辑器实时更新（没有它 words_delta 永远上报 0）
watch(() => props.currentWordCount, (n) => {
  // 基线还没初始化（刚开计时/刚切章）先立基线，不产生增量
  if (curBaseline.value < 0) { curBaseline.value = n }
  lastWordCount.value = n
})
// 切章/切文档：结算当前章节未落账的增量，基线重置为待初始化（下一拍字数进来时立新基线）
watch(() => props.activeDocId, () => {
  if (curBaseline.value >= 0) {
    settledWords.value += lastWordCount.value - curBaseline.value
    curBaseline.value = -1
  }
})

function resetWordBaseline() {
  settledWords.value = 0
  curBaseline.value = props.currentWordCount
}

function tick() {
  now.value = Date.now()
  if (state.value === 'running') checkMilestones()
  saveSnapshot()
}

// ---- 后端交互 ----
const myNickname = ref('') // 心跳携带，供对方卡片显示昵称
let beatMs = 0 // 当前心跳间隔；0=未启动
function scheduleBeat(ms: number) {
  if (beatRef.value && beatMs === ms) return
  if (beatRef.value) clearInterval(beatRef.value)
  beatRef.value = setInterval(heartbeat, ms)
  beatMs = ms
}
async function heartbeat() {
  if (state.value !== 'running' && state.value !== 'paused') return
  try {
    const res = await buddyApi.heartbeat({
      nickname: myNickname.value,
      project_id: Number(localStorage.getItem('xzj-active-project') || 0),
      words_delta: wordsDelta.value,
      elapsed_sec: Math.floor(elapsedMs.value / 1000),
      timer_state: state.value,
    })
    if (res.data?.data?.server_time) {
      clockOffset = res.data.data.server_time - Date.now()
    }
    applyToday(res.data?.data)
    applyState(res.data?.data)
  } catch { /* 网络断就下一拍再试 */ }
}

function applyState(d: any) {
  if (!d) return
  const s = d.state || {}
  waiting.value = !!s.waiting
  applyPeer(s.peer || null)
  // 排队等待时加快轮询（3s），配上/取消后回到 15s
  if (state.value === 'running' || state.value === 'paused') {
    scheduleBeat(waiting.value ? 3000 : 15000)
  }
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
      applyPeer(res.data.data.peer)
    } else {
      waiting.value = true
    }
  } catch { }
}
async function leaveMatch() {
  cancelRematch()
  try { await buddyApi.leaveMatch() } catch { }
  waiting.value = false
  applyPeer(null)
}
async function cheer() {
  try {
    await buddyApi.cheer()
    cheerJustSent.value = true
    setTimeout(() => { cheerJustSent.value = false }, 2000)
  } catch { }
}

// ---- SSE 实时事件（配对成功/对方离开/加油即时到达，不等心跳）----
let esAbort: AbortController | null = null
async function openEvents() {
  closeEvents()
  const ac = new AbortController()
  esAbort = ac
  try {
    const res = await fetch('/api/buddy/events', {
      headers: { Authorization: `Bearer ${localStorage.getItem('access_token') || ''}` },
      signal: ac.signal,
    })
    if (!res.ok || !res.body) throw new Error('sse bad response')
    const reader = res.body.getReader()
    const dec = new TextDecoder()
    let buf = ''
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      buf += dec.decode(value, { stream: true })
      let i: number
      while ((i = buf.indexOf('\n\n')) >= 0) {
        const chunk = buf.slice(0, i)
        buf = buf.slice(i + 2)
        const line = chunk.split('\n').find((l) => l.startsWith('data: '))
        if (line) { try { handleEvent(JSON.parse(line.slice(6))) } catch { } }
      }
    }
  } catch { /* 主动关闭或网络错误都走这里 */ }
  // 非主动关闭且仍在计时 → 3s 后重连
  if (esAbort === ac && !ac.signal.aborted &&
    (state.value === 'running' || state.value === 'paused')) {
    setTimeout(() => { if (esAbort === ac) openEvents() }, 3000)
  }
}
function closeEvents() {
  if (esAbort) { const ac = esAbort; esAbort = null; ac.abort() }
}
function handleEvent(ev: any) {
  if (!ev || !ev.type) return
  if (ev.type === 'matched') {
    waiting.value = false
    cancelRematch(false)
    if (ev.peer) applyPeer(ev.peer)
    scheduleBeat(15000)
  } else if (ev.type === 'peer_left') {
    applyPeer(null)
    scheduleRematch()
  } else if (ev.type === 'cheer') {
    peerCheerFlash.value = true
    setTimeout(() => { peerCheerFlash.value = false }, 4000)
  } else if (ev.type === 'peer_milestone') {
    if (ev.peer) applyPeer({ ...peer.value!, ...ev.peer })
    flashPeerMilestone(`${ev.peer?.nickname || '搭子'} 今日破 ${ev.milestone} 字了！`)
  } else if (ev.type === 'peer_nudge') {
    const icon = NUDGE_ICONS[ev.nudge] || '✦'
    peerNudgeFlash.value = `${icon} ${ev.from || '搭子'} ${NUDGE_RECV[ev.nudge] || '戳了戳你'}`
    if (peerNudgeTimer) clearTimeout(peerNudgeTimer)
    peerNudgeTimer = setTimeout(() => { peerNudgeFlash.value = '' }, 4000)
  }
}

// ---- 对方离开后自动重排队（P1）：30s 倒计时，可取消；只在计时中生效 ----
const rematchCountdown = ref(0)
let rematchTimer: ReturnType<typeof setInterval> | null = null
function scheduleRematch() {
  if (state.value !== 'running' && state.value !== 'paused') return
  cancelRematch(false)
  rematchCountdown.value = 30
  rematchTimer = setInterval(() => {
    rematchCountdown.value -= 1
    if (rematchCountdown.value <= 0) {
      cancelRematch(false)
      waiting.value = true
      match()
    }
  }, 1000)
}
function cancelRematch(clearUI = true) {
  if (rematchTimer) { clearInterval(rematchTimer); rematchTimer = null }
  if (clearUI) rematchCountdown.value = 0
}
// 模板点击入口（事件对象误传进 clearUI）
const onCancelRematchClick = () => cancelRematch(true)

// ---- 计时器逻辑（同 WritingTimer，含持久化） ----
function saveSnapshot() {
  const snap = {
    state: state.value, accumulatedMs: accumulatedMs.value, sessionStart: sessionStart.value,
    startWordCount: startWordCount.value, lastWordCount: lastWordCount.value,
    settledWords: settledWords.value, curBaseline: curBaseline.value,
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
    if (typeof s.settledWords === 'number') settledWords.value = s.settledWords
    if (typeof s.curBaseline === 'number') curBaseline.value = s.curBaseline
    summaryMs.value = s.summaryMs
    summaryWords.value = s.summaryWords
    // 恢复运行/暂停态后必须重启心跳，否则配对状态永远不同步、服务端 40s 后判离线
    if (state.value === 'running' || state.value === 'paused') {
      scheduleBeat(15000)
      heartbeat()
      openEvents()
      refreshToday().then(() => {
        // 里程碑档位以服务端今日统计为基线，防止恢复后重复弹花
        msWordLevel = Math.floor(Math.max(0, todayStat.value.words) / MS_WORD_STEP)
        msTimeLevel = Math.floor(Math.max(0, todayStat.value.ms) / MS_TIME_STEP_MS)
      })
    }
  } catch { }
}
function clearSnapshot() { try { localStorage.removeItem(STORAGE_KEY) } catch { } }

function start() {
  if (state.value === 'idle' || state.value === 'summary') {
    // 新一场：从 00:00 开始（总结态点 ▶ 也是新一场，不再续旧时长）
    accumulatedMs.value = 0
    startWordCount.value = props.currentWordCount
    lastWordCount.value = props.currentWordCount
    resetWordBaseline()
    msWordLevel = Math.floor(Math.max(0, todayStat.value.words) / MS_WORD_STEP)
    msTimeLevel = Math.floor(Math.max(0, todayStat.value.ms) / MS_TIME_STEP_MS)
  }
  sessionStart.value = Date.now()
  state.value = 'running'
  if (intervalRef.value) clearInterval(intervalRef.value)
  intervalRef.value = setInterval(tick, 1000)
  scheduleBeat(15000)
  heartbeat()
  openEvents()
  refreshToday()
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
  summaryWords.value = wordsDelta.value // 跨章口径：已结算 + 当前章节未结算
  state.value = 'summary'
  cancelRematch()
  leaveMatch()
  if (beatRef.value) { clearInterval(beatRef.value); beatRef.value = null; beatMs = 0 }
  closeEvents()
  buddyApi.stop().catch(() => { })
  refreshToday() // 总结条带今日累计
  saveSnapshot()
}
function reset() {
  state.value = 'idle'
  accumulatedMs.value = 0
  startWordCount.value = 0
  lastWordCount.value = 0
  settledWords.value = 0
  curBaseline.value = 0
  summaryMs.value = 0
  summaryWords.value = 0
  myMilestoneFlash.value = null
  peerMilestoneFlash.value = ''
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
  // 拉一次昵称供心跳携带（对方卡片显示用）
  userApi.me().then((res) => {
    myNickname.value = (res.data?.data || res.data)?.nickname || ''
  }).catch(() => { })
})
onBeforeUnmount(() => {
  saveSnapshot()
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
  if (intervalRef.value) clearInterval(intervalRef.value)
  if (beatRef.value) clearInterval(beatRef.value)
  if (nudgeTick) clearInterval(nudgeTick)
  if (rematchTimer) clearInterval(rematchTimer)
  if (msFlashTimer) clearTimeout(msFlashTimer)
  if (peerMsTimer) clearTimeout(peerMsTimer)
  if (peerNudgeTimer) clearTimeout(peerNudgeTimer)
  closeEvents()
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
/* 我的字数=绿胶囊（阵营色：我绿 / TA 蓝） */
.me-chip {
  display: inline-flex; align-items: center; gap: 2px;
  font-size: 10.5px; font-weight: 600; color: #2f9e44;
  background: rgba(48, 209, 88, 0.13); padding: 1px 7px; border-radius: 99px;
  font-variant-numeric: tabular-nums;
}
.me-chip.negative, .me-chip.zero { color: var(--text-tertiary, #999); background: var(--bg-hover, rgba(0,0,0,0.06)); }
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
.who { font-size: 9px; color: #fff; background: #81bb8e; border-radius: 4px; padding: 0 4px; margin-right: 4px; vertical-align: 1px; }
.who.ta { background: #4a9bf5; }
.bw-peer-stat { display: flex; gap: 6px; font-size: 10.5px; color: var(--text-tertiary, #999); font-variant-numeric: tabular-nums; }
/* TA 侧=蓝色系（阵营色），时间加粗、字数胶囊 */
.bw-peer-time { color: var(--accent, #007aff); font-weight: 600; }
.bw-peer-words { color: var(--accent, #007aff); background: rgba(0, 122, 255, 0.1); padding: 1px 6px; border-radius: 99px; }
.bw-peer-words.revise { color: var(--text-tertiary, #999); background: var(--bg-hover, rgba(0,0,0,0.06)); }
.peer-paused-tag { margin-left: 4px; font-size: 9px; color: var(--text-tertiary, #999); }
.bw-today-line {
  margin-top: 5px; padding-top: 6px; border-top: 1px dashed var(--border-subtle, rgba(0,0,0,0.06));
  text-align: center; font-size: 10.5px; color: var(--text-tertiary, #999); font-variant-numeric: tabular-nums;
}
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

/* ---- 里程碑 ---- */
.bw-ms-strip {
  display: flex; align-items: center; gap: 4px; margin-top: 8px;
  padding-top: 7px; border-top: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
}
.bw-ms-strip .ms-tick { width: 12px; height: 3px; border-radius: 2px; background: var(--bg-hover, rgba(0,0,0,0.12)); }
.bw-ms-strip .ms-tick.on { background: #30d158; }
.bw-ms-strip .ms-tick.now { background: var(--accent, #007aff); animation: bfab 1.2s infinite; }
.bw-ms-strip .ms-next { margin-left: auto; font-size: 9px; color: var(--text-tertiary, #999); white-space: nowrap; }

.bw-ms-flash {
  margin-top: 7px; padding: 5px 8px; border-radius: 8px; text-align: center;
  font-size: 11px; font-weight: 600; animation: cheerin 0.4s ease;
}
.bw-ms-flash.peer {
  background: rgba(10,132,255,0.1); color: var(--accent, #007aff);
}

/* ---- 快捷互动条 ---- */
.bw-nudge-flash {
  margin-top: 7px; padding: 5px 8px; border-radius: 8px; text-align: center;
  background: rgba(48,209,88,0.1); color: #2f9e44; font-size: 11px; font-weight: 600;
  animation: cheerin 0.4s ease;
}
.bw-nudge-row {
  display: flex; gap: 4px; margin-top: 7px; padding-top: 7px;
  border-top: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
}
.bw-nudge-chip {
  flex: 1; border: none; border-radius: 7px; padding: 5px 0; cursor: pointer;
  background: var(--bg-hover, rgba(0,0,0,0.06)); color: var(--text-secondary, #666);
  font-size: 9.5px; font-weight: 600; white-space: nowrap;
  transition: background 0.15s ease, transform 0.1s ease;
}
.bw-nudge-chip:hover { background: rgba(0,122,255,0.12); color: var(--accent, #007aff); }
.bw-nudge-chip:active { transform: scale(0.94); }
.bw-nudge-chip.cd { opacity: 0.35; pointer-events: none; }

.bw-ms-pop {
  position: fixed; left: 50%; top: 38%; transform: translate(-50%, -50%);
  z-index: 999; pointer-events: none;
  display: flex; flex-direction: column; align-items: center; gap: 1px;
  padding: 14px 28px; border-radius: 16px; text-align: center;
  background: var(--bg-secondary, #fff);
  box-shadow: 0 12px 48px rgba(0,0,0,0.22), 0 0 0 1px rgba(0,0,0,0.06);
  animation: mpop 0.45s cubic-bezier(0.2, 1.4, 0.4, 1), mfade 0.5s ease 2.5s forwards;
}
@keyframes mpop { from { transform: translate(-50%, -35%) scale(0.7); opacity: 0; } }
@keyframes mfade { to { opacity: 0; transform: translate(-50%, -55%); } }
.bw-ms-pop .ms-emoji { font-size: 20px; color: var(--accent, #007aff); }
.bw-ms-pop .ms-title { font-size: 14px; font-weight: 800; color: var(--accent, #007aff); }
.bw-ms-pop .ms-sub { font-size: 10.5px; color: var(--text-tertiary, #999); }

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
.bw-summary .sum-close { display: none; font-size: 10px; color: var(--text-tertiary, #999); }
.bw-summary:hover .sum-close { display: inline; }
.sum-time { font-size: 15px; font-weight: 700; color: var(--accent, #007aff); font-variant-numeric: tabular-nums; }
.sum-sep { color: var(--text-tertiary, #999); }
.sum-words { font-size: 12px; color: #30d158; }
.sum-words.negative { color: var(--text-tertiary, #999); }

@media (prefers-color-scheme: dark) {
  .buddy-window, .buddy-fab { background: #2c2c2e; color: #e5e5ea; }
  .bw-title, .bw-peer-name { color: #e5e5ea; }
  .me-chip { background: rgba(48, 209, 88, 0.2); }
  .bw-peer-words { background: rgba(0, 122, 255, 0.25); }
}
</style>
