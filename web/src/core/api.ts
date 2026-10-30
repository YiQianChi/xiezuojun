import axios from 'axios'

// ponytail-harmony: 桌面/网页=相对路径；鸿蒙壳=本地 rawfile 加载，API 需绝对地址（构建时注入）
const API_BASE = (import.meta as any).env?.VITE_API_BASE ?? ''
const api = axios.create({
  baseURL: API_BASE + '/api',
  timeout: 10000,
})

// 请求拦截器：自动带 token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Token 刷新状态
let isRefreshing = false
let pendingQueue: Array<{ resolve: (v: string) => void; reject: (e: unknown) => void }> = []

function onRefreshed(token: string) {
  pendingQueue.forEach(p => p.resolve(token))
  pendingQueue = []
}

function onRefreshFailed(err: unknown) {
  pendingQueue.forEach(p => p.reject(err))
  pendingQueue = []
}

// 响应拦截器：401 自动刷新 token，失败才跳登录
api.interceptors.response.use(
  res => res,
  async err => {
    const originalRequest = err.config

    // 非 401 或已重试过，直接 reject
    if (!err.response || err.response.status !== 401 || originalRequest._retried) {
      return Promise.reject(err)
    }

    // /auth/refresh 自身 401 = refresh token 也过期了，跳登录
    if (originalRequest.url === '/auth/refresh') {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login'
      }
      return Promise.reject(err)
    }

    // 正在刷新：排队等待
    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        pendingQueue.push({
          resolve: (token: string) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            originalRequest._retried = true
            resolve(api(originalRequest))
          },
          reject,
        })
      })
    }

    // 开始刷新
    isRefreshing = true
    originalRequest._retried = true
    const refreshToken = localStorage.getItem('refresh_token')

    if (!refreshToken) {
      isRefreshing = false
      localStorage.removeItem('access_token')
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login'
      }
      return Promise.reject(err)
    }

    try {
      // ponytail-harmony: 刷新请求也走 API_BASE（鸿蒙壳下同域绝对地址）
      const res = await axios.post(API_BASE + '/api/auth/refresh', { refresh_token: refreshToken })
      const newToken = res.data.data?.access_token || res.data.access_token
      const newRefresh = res.data.data?.refresh_token || res.data.refresh_token
      localStorage.setItem('access_token', newToken)
      if (newRefresh) localStorage.setItem('refresh_token', newRefresh)

      onRefreshed(newToken)
      isRefreshing = false

      originalRequest.headers.Authorization = `Bearer ${newToken}`
      return api(originalRequest)
    } catch (refreshErr) {
      onRefreshFailed(refreshErr)
      isRefreshing = false
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login'
      }
      return Promise.reject(refreshErr)
    }
  }
)

export default api

// 用户相关
export const userApi = {
  sendEmailCode(data: { email: string; scene: 'register' | 'login' | 'reset_password' }) {
    return api.post('/auth/email-code', data)
  },
  register(data: { email: string; password: string; code: string }) {
    return api.post('/auth/register', data)
  },
  login(data: { email: string; password?: string; code?: string }) {
    return api.post('/auth/login', data)
  },
  me() {
    return api.get('/user/me')
  },
  changePassword(data: { old_password: string; new_password: string }) {
    return api.post('/user/change-password', data)
  },
  updateProfile(data: { nickname?: string; avatar?: string }) {
    return api.patch('/user/me', data)
  },
  usage() {
    return api.get('/user/usage')
  },
  resetPassword(data: { email: string; code: string; password: string }) {
    return api.post('/auth/reset-password', data)
  },
}

// 项目相关
export const projectApi = {
  list() {
    return api.get('/projects')
  },
  get(id: string | number) {
    return api.get(`/projects/${id}`)
  },
  create(data: { name: string; description?: string; first_chapter_title?: string; setting_types?: string[] }) {
    return api.post('/projects', data)
  },
  importBook(data: { name: string; volumes: { title: string; chapters: { title: string; content: string }[] }[] }) {
    // 大书（几百万字）传输+落库要几分钟，全局 10s 超时会掐断 → 单独放宽
    return api.post('/projects/import', data, { timeout: 600000 })
  },
  importBookBatch(data: { project_id: number; name: string; volumes: { title: string; chapters: { title: string; content: string }[] }[] }) {
    return api.post('/projects/import/batch', data, { timeout: 600000 })
  },
  update(id: string | number, data: { name?: string; description?: string }) {
    return api.patch(`/projects/${id}`, data)
  },
  delete(id: string) {
    return api.delete(`/projects/${id}`)
  },
  stats(id: string | number) {
    return api.get(`/projects/${id}/stats`)
  },
  exportProject(id: string | number) {
    return api.get(`/projects/${id}/export`)
  },
  exportEPUB(id: string | number) {
    return api.get(`/projects/${id}/export/epub`, { responseType: 'blob' })
  },
  exportTXT(id: string | number) {
    return api.get(`/projects/${id}/export/txt`, { responseType: 'blob' })
  },
  exportDOCX(id: string | number) {
    return api.get(`/projects/${id}/export/docx`, { responseType: 'blob' })
  },
}

// 项目内搜索
export const searchApi = {
  documents(projectId: string | number, q: string, limit = 200) {
    return api.get(`/projects/${projectId}/search`, { params: { q, limit } })
  },
}

// 文档相关
export const documentApi = {
  list(projectId: string, branchId?: number | string | null) {
    return api.get('/documents', { params: { project_id: projectId, branch_id: branchId || undefined } })
  },
  create(data: { project_id: number; branch_id?: number; title: string; type: string; parent_id?: number; sort_order?: number; content?: string; icon?: string }) {
    return api.post('/documents', data)
  },
  get(id: string | number) {
    return api.get(`/documents/${id}`)
  },
  update(id: string | number, data: { title?: string; content?: string; create_version?: boolean; version_label?: string; ai_include?: boolean; icon?: string; doc_status?: string; word_goal?: number; meta?: string }) {
    return api.patch(`/documents/${id}`, data)
  },
  listVersions(id: string | number) {
    return api.get(`/documents/${id}/versions`)
  },
  getVersion(versionId: string | number) {
    return api.get(`/document-versions/${versionId}`)
  },
  restoreVersion(versionId: string | number) {
    return api.post(`/document-versions/${versionId}/restore`)
  },
  move(id: string | number, data: { parent_id?: string | null; sort_order?: number }) {
    return api.patch(`/documents/${id}/move`, data)
  },
  delete(id: string | number) {
    return api.delete(`/documents/${id}`)
  },
}

// AI 相关
export const aiApi = {
  generate(data: { project_id: string; document_id?: string; mode: string; content: string; stream?: boolean }) {
    return api.post('/ai/generate', data, { timeout: 120000 })
  },
  chat(data: { project_id: string; messages: { role: string; content: string }[]; stream?: boolean }) {
    return api.post('/ai/chat', data, { timeout: 120000 })
  },
}

// TTS 朗读
export const ttsApi = {
  voices() {
    return api.get('/tts/voices')
  },
  synthesize(text: string, voice: string) {
    return api.post('/tts', { text, voice }, { responseType: 'blob', timeout: 30000 })
  },
}

// 记忆系统
export const memoryApi = {
  listCharacters(projectId: string) {
    return api.get('/memory/characters', { params: { project_id: projectId } })
  },
  getCharacter(projectId: string, name: string) {
    return api.get(`/memory/characters/${encodeURIComponent(name)}`, { params: { project_id: projectId } })
  },
  writeCharacter(projectId: string, name: string, content: string) {
    return api.put(`/memory/characters/${encodeURIComponent(name)}`, { content }, { params: { project_id: projectId } })
  },
  deleteCharacter(projectId: string, name: string) {
    return api.delete(`/memory/characters/${encodeURIComponent(name)}`, { params: { project_id: projectId } })
  },
  listSummaries(projectId: string) {
    return api.get('/memory/summaries', { params: { project_id: projectId } })
  },
  getSummary(projectId: string, chapterId: string) {
    return api.get(`/memory/summaries/${chapterId}`, { params: { project_id: projectId } })
  },
  getFile(projectId: string, path: string) {
    return api.get('/memory/file', { params: { project_id: projectId, path } })
  },
  writeFile(projectId: string, path: string, content: string) {
    return api.put('/memory/file', { content }, { params: { project_id: projectId, path } })
  },
  getRelationships(projectId: string) {
    return api.get('/memory/relationships', { params: { project_id: projectId } })
  },
  saveRelationships(projectId: string, data: any) {
    return api.put('/memory/relationships', data, { params: { project_id: projectId } })
  },
}

// 回收站
export const trashApi = {
  list(projectId?: string | number) {
    const params = projectId ? { project_id: projectId } : {}
    return api.get('/trash', { params })
  },
  restore(data: { project_id: number; doc_id: number }) {
    return api.post('/trash/restore', data)
  },
  empty(projectId?: string | number) {
    const params = projectId ? { project_id: projectId } : {}
    return api.delete('/trash', { params })
  },
  destroy(data: { project_id: number; doc_id: number }) {
    return api.post('/trash/destroy', data)
  },
}

// 伏笔/钩子追踪
export interface StoryThread {
  id?: number
  project_id: number
  type: 'foreshadowing' | 'hook' | 'promise' | 'suspense'
  title: string
  description?: string
  status: 'open' | 'resolved' | 'abandoned'
  planted_chapter?: number | null
  resolved_chapter?: number | null
  urgency?: number
  sort_order?: number
}

export const threadApi = {
  list(projectId: string | number) {
    return api.get(`/projects/${projectId}/threads`)
  },
  create(projectId: string | number, data: Partial<StoryThread>) {
    return api.post(`/projects/${projectId}/threads`, { ...data, project_id: Number(projectId) })
  },
  update(id: string | number, data: Partial<StoryThread>) {
    return api.patch(`/threads/${id}`, data)
  },
  delete(id: string | number) {
    return api.delete(`/threads/${id}`)
  },
}

// 搭子（结伴写作）
export const buddyApi = {
  heartbeat(data: { nickname?: string; project_id: number; words_delta: number; elapsed_sec?: number; timer_state?: string }) {
    return api.post('/buddy/heartbeat', data)
  },
  stop() {
    return api.post('/buddy/stop')
  },
  finish(data: { project_id: number; started_at: number; ended_at: number; duration_sec: number; words_delta: number }) {
    return api.post('/buddy/finish', data)
  },
  sessionStats(projectId?: number) {
    return api.get('/buddy/session-stats', { params: projectId ? { project_id: projectId } : {} })
  },
  report() {
    return api.get('/buddy/report')
  },
  match() {
    return api.post('/buddy/match')
  },
  leaveMatch() {
    return api.post('/buddy/leave-match')
  },
  cheer() {
    return api.post('/buddy/cheer')
  },
  nudge(nudge: string) {
    return api.post('/buddy/nudge', { nudge })
  },
  state() {
    return api.get('/buddy/state', { params: { consume_cheer: 1 } })
  },
  today() {
    return api.get('/buddy/today')
  },
}

// 拼字房（多人拼字）
export const sprintApi = {
  create(data: { nickname?: string; name: string; mode: 'time' | 'words'; target: number; is_public: boolean; password?: string; max_players: number }) {
    return api.post('/sprint/create', data)
  },
  join(data: { nickname?: string; code: string; password?: string }) {
    return api.post('/sprint/join', data)
  },
  quickmatch(nickname?: string) {
    return api.post('/sprint/quickmatch', { nickname })
  },
  leave() {
    return api.post('/sprint/leave')
  },
  start() {
    return api.post('/sprint/start')
  },
  heartbeat(words: number) {
    return api.post('/sprint/heartbeat', { words })
  },
  room() {
    return api.get('/sprint/room')
  },
}

// 后台管理（仅 is_admin 账号可用，服务端校验）
export const adminApi = {  statsOverview() { return api.get('/admin/stats/overview') },
  statsDaily() { return api.get('/admin/stats/daily') },
  statsAIUsage() { return api.get('/admin/stats/ai-usage') },
  listUsers(params: { q?: string; page?: number }) { return api.get('/admin/users', { params }) },
  getUser(userId: string) { return api.get(`/admin/users/${userId}`) },
  setUserStatus(userId: string, status: number) { return api.patch(`/admin/users/${userId}/status`, { status }) },
  setUserPlan(userId: string, plan: string) { return api.patch(`/admin/users/${userId}/plan`, { plan }) },
  resetUserPassword(userId: string, password: string) { return api.post(`/admin/users/${userId}/reset-password`, { password }) },
  updateUser(userId: string, data: Partial<{ nickname: string; phone: string; email: string; plan: string; status: number; is_admin: boolean }>) { return api.patch(`/admin/users/${userId}`, data) },
  listProjects(params: { q?: string; page?: number }) { return api.get('/admin/projects', { params }) },
  listProjectDocuments(projectId: number | string) { return api.get(`/admin/projects/${projectId}/documents`) },
  getDocumentContent(docId: number | string) { return api.get(`/admin/documents/${docId}/content`) },
  setProjectStatus(projectId: number | string, status: number) { return api.patch(`/admin/projects/${projectId}/status`, { status }) },
  listAnnouncements() { return api.get('/admin/announcements') },
  createAnnouncement(data: { title: string; content: string }) { return api.post('/admin/announcements', data) },
  deleteAnnouncement(id: number) { return api.delete(`/admin/announcements/${id}`) },
  listFeedbacks(params?: { open?: number }) { return api.get('/admin/feedbacks', { params }) },
  resolveFeedback(id: number) { return api.patch(`/admin/feedbacks/${id}`, {}) },
}
