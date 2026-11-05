// 原生工作台（首页）：自定义导航自绘标题栏，返回/主页跟标题一体
const api = require('../../utils/api')

const QUOTES = [
  '「夜还长，茶还热，故事才写到一半。」',
  '「写不出来的时候，先把上一段再读一遍。」',
  '「所有的初稿都是臭的，先写完再说。」',
  '「人物活了，情节自己会走。」',
  '「今天写五百字，也是没有断更的一天。」',
  '「别怕写坏，怕的是没写。」',
  '「灵感是等不来的，是写出来的。」'
]

Page({
  data: {
    sbh: 44,          // 状态栏高
    capsuleRight: 87, // 胶囊左缘距屏右（标题栏右侧留白）
    quote: '',
    nickname: '',
    projects: [],
    recent: null,
    showCreate: false,
    newName: '',
    creating: false
  },

  onLoad() {
    const sbh = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()).statusBarHeight || 44
    let capsuleRight = 87
    try {
      const m = wx.getMenuButtonBoundingClientRect()
      capsuleRight = m && m.width ? Math.round(m.width + (m.left >= 0 ? (wx.getSystemInfoSync().windowWidth - m.right) : 10) + 10) : 87
    } catch (e) {}
    this.setData({
      sbh,
      capsuleRight,
      quote: QUOTES[Math.floor(Date.now() / 86400000) % QUOTES.length]
    })
  },

  onShow() {
    // 每次回到首页刷新（从编辑器返回能看到最新最近打开）
    const at = wx.getStorageSync('xzj_token')
    if (!at) {
      wx.reLaunch({ url: '/pages/index/index' }) // 未登录 → 游客营销页（web-view）
      return
    }
    this.load()
  },

  async load() {
    try {
      const res = await api.get('/user/me')
      const me = res.data || res
      this.setData({ nickname: (me.nickname || '我').slice(0, 6) })
    } catch (e) { /* 昵称拿不到不阻塞 */ }
    try {
      const res = await api.get('/projects')
      const list = (res.data || res || []).map(p => ({ ...p, cover_url: api.coverUrl(p.cover_url) }))
      const sorted = [...list].sort((a, b) => +new Date(b.updated_at) - +new Date(a.updated_at))
      this.setData({ projects: sorted, recent: sorted[0] || null })
    } catch (e) { /* 401 已由 api 层处理 */ }
  },

  openRecent(e) {
    const pid = e.currentTarget.dataset.pid
    if (pid) api.openEditor(pid)
  },

  // 开始写作：有最近的书 → 接着写；没有 → 弹新建
  startWriting() {
    if (this.data.recent) api.openEditor(this.data.recent.public_id || String(this.data.recent.id))
    else this.openCreate()
  },

  openCreate() {
    this.setData({ showCreate: true, newName: '' })
  },
  onName(e) { this.setData({ newName: e.detail.value }) },
  cancelCreate() { this.setData({ showCreate: false }) },

  async createProject() {
    const name = this.data.newName.trim()
    if (!name || this.data.creating) return
    this.setData({ creating: true })
    try {
      const res = await api.post('/projects', { name, description: '', setting_types: [] })
      const p = res.data || res
      this.setData({ showCreate: false, creating: false })
      api.openEditor(p.public_id || String(p.id))
    } catch (e) {
      this.setData({ creating: false })
      wx.showToast({ title: (e.data && e.data.error) || '创建失败', icon: 'none' })
    }
  },

  goShelf() { wx.navigateTo({ url: '/pages/shelf/shelf' }) },
  goImport() {
    // 导入走网页书架的导入面板（文件选择/解析逻辑网页已有）
    const at = wx.getStorageSync('xzj_token')
    const rt = wx.getStorageSync('xzj_rt')
    let url = '/pages/index/index?web=bookshelf%3Fimport%3D1'
    if (at) url += '&xzj_at=' + encodeURIComponent(at)
    if (rt) url += '&xzj_rt=' + encodeURIComponent(rt)
    wx.navigateTo({ url })
  }
})
