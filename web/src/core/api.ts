import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
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

// 响应拦截器：401 自动跳转登录
api.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401 && !window.location.pathname.includes('/login')) {
      localStorage.removeItem('access_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api

// 用户相关
export const userApi = {
  sendEmailCode(data: { email: string; scene: 'register' | 'login' }) {
    return api.post('/auth/email-code', data)
  },
  register(data: { email: string; password: string; code: string }) {
    return api.post('/auth/register', data)
  },
  login(data: { email: string; password?: string; code?: string }) {
    return api.post('/auth/login', data)
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
  create(data: { name: string; description?: string }) {
    return api.post('/projects', data)
  },
  delete(id: string) {
    return api.delete(`/projects/${id}`)
  },
}

// 故事线相关
export const branchApi = {
  list(projectId: string | number) {
    return api.get(`/projects/${projectId}/branches`)
  },
  create(projectId: string | number, data: { name: string; base_branch_id?: number; fork_document_id: number }) {
    return api.post(`/projects/${projectId}/branches`, data)
  },
  update(branchId: string | number, data: { name?: string; status?: number }) {
    return api.patch(`/story-branches/${branchId}`, data)
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
  create(data: { project_id: number; branch_id?: number; title: string; type: string; parent_id?: number; sort_order?: number; content?: string }) {
    return api.post('/documents', data)
  },
  get(id: string | number) {
    return api.get(`/documents/${id}`)
  },
  update(id: string | number, data: { title?: string; content?: string; create_version?: boolean; version_label?: string }) {
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
}