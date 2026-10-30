// 拼字房状态（useToast 同款模块级单例）：房间快照 + SSE + 心跳上报 + HUD 派生数据。
// 形态=C 方案（沉浸横幅）：拼字中正文照常写，顶部细横幅 + 右缘迷你榜；
// 专注模式降级为右上倒计时胶囊 + 底部淡字（HUD 渲染在 Editor.vue，弹窗在 SprintPanel.vue）。
import { ref, computed, reactive, type Ref } from 'vue'
import { sprintApi } from '../core/api'
import { API_BASE } from '../core/sse'
import { toast } from './useToast'

export interface SprintPlayer {
  user_id: string
  nickname: string
  words: number
  online: boolean
  rank: number
  is_host: boolean
}

export interface SprintRoom {
  code: string
  name: string
  mode: string
  target: number
  target_label: string
  is_public: boolean
  has_password: boolean
  max_players: number
  host_uid: string
  state: string // waiting | running | finished
  started_at_ms: number
  ends_at_ms: number
  server_now_ms: number
  players: SprintPlayer[]
  me: string
}

// ---- 单例状态 ----
const room = ref<SprintRoom | null>(null)
const panelOpen = ref(false)
const phase = ref<'entry' | 'create' | 'join' | 'lobby'>('entry')
const joinCode = ref('')
const joinPass = ref('')
const busy = ref(false)
const showResult = ref(false)
const now = ref(Date.now())
let clockOffset = 0
let wordCountRef: Ref<number> | null = null
let beatTimer: number | null = null
let esAbort: AbortController | null = null
let lastForm: { name: string; mode: 'time' | 'words'; target: number; is_public: boolean; password: string; max_players: number } | null = null

// 每秒 tick：驱动倒计时显示
setInterval(() => { now.value = Date.now() + clockOffset }, 1000)

function nickname(): string {
  return localStorage.getItem('xzj-nickname') || '菌友'
}

function applyRoom(r: SprintRoom | null) {
  if (!r) return
  const prev = room.value
  room.value = r
  clockOffset = r.server_now_ms - Date.now()
  if (r.state === 'finished' && (!prev || prev.state !== 'finished')) {
    showResult.value = true
  }
  if (r.state === 'waiting' && phase.value !== 'lobby') phase.value = 'lobby'
}

// ---- SSE：房内事件（快照/开局/结算/解散）实时到达 ----
async function openEvents() {
  closeEvents()
  const ac = new AbortController()
  esAbort = ac
  try {
    const res = await fetch(API_BASE + '/api/sprint/events', {
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
        if (line) { try { handleEvent(JSON.parse(line.slice(6))) } catch { /* 忽略坏帧 */ } }
      }
    }
  } catch { /* 主动关闭/网络错误 */ }
  // 仍在房 → 3s 重连
  if (esAbort === ac && !ac.signal.aborted && room.value) {
    setTimeout(() => { if (esAbort === ac) openEvents() }, 3000)
  }
}
function closeEvents() {
  if (esAbort) { const ac = esAbort; esAbort = null; ac.abort() }
}
function handleEvent(ev: any) {
  if (!ev || !ev.type) return
  if (ev.type === 'room' && ev.room) applyRoom(ev.room)
  else if (ev.type === 'finished' && ev.room) applyRoom(ev.room)
  else if (ev.type === 'dissolved') {
    room.value = null
    phase.value = 'entry'
    panelOpen.value = false
    toast.info('房间已解散（房主离开）')
  }
}

// ---- 心跳：约 5s 一拍，上报本局字数（Editor 传入的 sessionWords 已按章折叠，服务端广播榜单） ----
function startBeat() {
  stopBeat()
  beatTimer = window.setInterval(async () => {
    if (!room.value || !wordCountRef) return
    const st = room.value.state
    if (st !== 'waiting' && st !== 'running') return
    const words = st === 'running' ? Math.max(0, wordCountRef.value) : 0
    try {
      const res = await sprintApi.heartbeat(words)
      const r = res.data?.data?.room
      if (r) applyRoom(r)
    } catch { /* 断网由 SSE 重连与下一拍兜底 */ }
  }, 5000)
}
function stopBeat() {
  if (beatTimer) { clearInterval(beatTimer); beatTimer = null }
}

// ---- 派生数据（HUD 用） ----
const inRoom = computed(() => !!room.value)
const running = computed(() => room.value?.state === 'running')
const meP = computed(() => room.value?.players.find(p => p.user_id === (room.value?.me || '')) || null)
const myRank = computed(() => meP.value?.rank || 0)
const myWords = computed(() => meP.value?.words || 0)
const leaderP = computed(() => room.value?.players[0] || null)
const iAmLeader = computed(() => !!leaderP.value && leaderP.value.user_id === room.value?.me)
const timeLeftSec = computed(() => {
  const r = room.value
  if (!r || r.mode !== 'time' || !r.ends_at_ms) return 0
  return Math.max(0, Math.round((r.ends_at_ms - now.value) / 1000))
})
const countdownText = computed(() => {
  const r = room.value
  if (!r) return ''
  if (r.mode === 'words') {
    const lead = leaderP.value?.words || 0
    return `🎯 ${Math.min(lead, r.target)}/${r.target}`
  }
  const s = timeLeftSec.value
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
})
const leadText = computed(() => {
  if (!leaderP.value) return ''
  return iAmLeader.value ? '我领先！' : `🏅 ${leaderP.value.nickname} ${leaderP.value.words} 字`
})
const amHost = computed(() => room.value?.host_uid === room.value?.me)

function avatar(name: string): string {
  return (name || '菌').trim().slice(0, 1)
}

// ---- 动作 ----
async function doCreate(form: { name: string; mode: 'time' | 'words'; target: number; is_public: boolean; password: string; max_players: number }) {
  busy.value = true
  try {
    const res = await sprintApi.create({ nickname: nickname(), ...form, password: form.is_public ? '' : form.password })
    applyRoom(res.data?.data)
    lastForm = { ...form }
    phase.value = 'lobby'
    openEvents(); startBeat()
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '创建失败，稍后再试')
  } finally { busy.value = false }
}

async function doJoin(code: string, password: string) {
  busy.value = true
  try {
    const res = await sprintApi.join({ nickname: nickname(), code: code.trim(), password })
    applyRoom(res.data?.data)
    phase.value = 'lobby'
    openEvents(); startBeat()
    return true
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '加入失败，检查房间号')
    return false
  } finally { busy.value = false }
}

async function doQuickmatch() {
  busy.value = true
  try {
    const res = await sprintApi.quickmatch(nickname())
    applyRoom(res.data?.data)
    phase.value = 'lobby'
    openEvents(); startBeat()
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '匹配失败，稍后再试')
  } finally { busy.value = false }
}

async function doLeave() {
  try { await sprintApi.leave() } catch { /* 房间可能已散 */ }
  room.value = null
  phase.value = 'entry'
  panelOpen.value = false
  showResult.value = false
  closeEvents(); stopBeat()
}

async function doStart() {
  busy.value = true
  try {
    const res = await sprintApi.start()
    applyRoom(res.data?.data)
    if (room.value?.state === 'running') panelOpen.value = false // 开拼自动收面板，回正文
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '开始失败')
  } finally { busy.value = false }
}

function again() {
  showResult.value = false
  if (lastForm) { phase.value = 'create'; panelOpen.value = true }
  else { phase.value = 'entry'; panelOpen.value = true }
}

// ---- 入口动作（深度融合：入口统一在搭子悬浮窗，通过事件让 BuddyWindow 开窗+切「拼字」页签）----
function focusSprintWindow() {
  window.dispatchEvent(new CustomEvent('xzj-sprint-focus'))
}
function openPanel() {
  focusSprintWindow()
}
function openJoin(code?: string) {
  phase.value = 'join'
  if (code) joinCode.value = code
  if (room.value) {
    toast.info('你还在拼字房里，先退出再加入新房间')
    phase.value = 'lobby'
  }
  focusSprintWindow()
}

// ---- 初始化（Editor setup 调一次；传「按章折叠的本局字数」computed；返回恢复的本局字数供 Editor 做种子） ----
let inited = false
async function init(wc: Ref<number>): Promise<number> {
  wordCountRef = wc
  if (inited) return 0
  inited = true
  let restored = 0
  // 断线/刷新恢复：先拉当前房间
  try {
    const res = await sprintApi.room()
    const r: SprintRoom | null = res.data?.data
    if (r) {
      if (r.state === 'running') {
        const mine = r.players.find(p => p.user_id === r.me)
        restored = mine?.words || 0
      }
      applyRoom(r)
      openEvents(); startBeat()
    }
  } catch { /* 未登录/网络错都静默 */ }
  // 邀请链接落地（App.vue 存的 ?sprint=code）
  const inv = localStorage.getItem('xzj-sprint-invite')
  if (inv) {
    localStorage.removeItem('xzj-sprint-invite')
    openJoin(inv)
  }
  return restored
}

export function useSprint() {
  return reactive({
    room, panelOpen, phase, joinCode, joinPass, busy, showResult,
    inRoom, running, amHost, myRank, myWords, leaderP, iAmLeader,
    countdownText, leadText, myPlayer: meP,
    avatar, init, openPanel, openJoin,
    create: doCreate, join: doJoin, quickmatch: doQuickmatch, leave: doLeave, start: doStart, again,
  })
}
