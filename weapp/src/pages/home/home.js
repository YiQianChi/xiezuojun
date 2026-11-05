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
    quote: '',
    nickname: '',
    projects: [],
    recent: null,
    showCreate: false,
    newName: '',
    creating: false
  },

  onLoad(options) {
    // 垫层转发：index 根层落地时 reLaunch 到这里，再 navigateTo 回目标 web-view 层（保证原生栏有返回箭头）
    if (options && options.forward) {
      this._forwarding = true // onShow 别抢跳（转发进行中）
      const fwd = decodeURIComponent(options.forward) + '&__fromHome=1'
      wx.navigateTo({ url: '/pages/index/index?' + fwd, fail: () => wx.reLaunch({ url: '/pages/index/index' }) })
      return
    }
    this.setData({ quote: QUOTES[Math.floor(Date.now() / 86400000) % QUOTES.length] })
  },

  onShow() {
    // 垫层转发进行中：onShow 先让路（游客读分享章的场景，避免被未登录跳转打断）
    if (this._forwarding) { this._forwarding = false; return }
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
    const item = this.data.projects[Number(e.currentTarget.dataset.idx)]
    if (item) this.openWrite(item)
  },

  // 打开书 → 原生码字模式（网页完整编辑器从码字页进）
  openWrite(item) {
    wx.navigateTo({ url: '/pages/write/write?pid=' + item.id + '&pub=' + encodeURIComponent(item.public_id || '') })
  },

  // 开始写作：有最近的书 → 接着写；没有 → 弹新建
  startWriting() {
    if (this.data.recent) this.openWrite(this.data.recent)
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
      this.openWrite({ id: p.id, public_id: p.public_id })
    } catch (e) {
      this.setData({ creating: false })
      wx.showToast({ title: (e.data && e.data.error) || '创建失败', icon: 'none' })
    }
  },

  goShelf() { wx.navigateTo({ url: '/pages/shelf/shelf' }) },
  goSettings() { wx.navigateTo({ url: '/pages/settings/settings' }) },

  // 原生导入：选 txt（从聊天记录）→ 解析卷/章 → 确认 → 分批入库
  async goImport() {
    let parsed
    try {
      parsed = await require('../../utils/importer').pickAndParse()
    } catch (e) {
      if (e.message !== 'cancel') wx.showToast({ title: e.message || '选择文件失败', icon: 'none' })
      return
    }
    const modeHint = parsed.mode === 'chapters'
      ? '识别到 ' + parsed.total + ' 章'
      : '未识别到章节标题，将按约 3000 字切分（' + parsed.total + ' 部分）'
    wx.showModal({
      title: '导入《' + parsed.name + '》',
      content: modeHint + '。确认导入吗？',
      editable: true,
      placeholderText: '书名（可改）',
      confirmText: '导入',
      success: r => {
        if (!r.confirm) return
        const bookName = (r.content || '').trim() || parsed.name
        wx.showLoading({ title: '导入中…', mask: true })
        require('../../utils/importer').runImport(bookName, parsed.vols, null)
          .then(out => {
            wx.hideLoading()
            wx.showToast({ title: '已导入 ' + out.total + ' 章', icon: 'none' })
            this.load()
          })
          .catch(e => {
            wx.hideLoading()
            wx.showToast({ title: (e.data && e.data.error) || '导入失败', icon: 'none' })
          })
      }
    })
  },

  // 退出小程序（官方 API；确认弹窗防误触）
  exitApp() {
    wx.showModal({
      title: '退出写作菌？',
      confirmText: '退出',
      confirmColor: '#d05252',
      success: r => { if (r.confirm) wx.exitMiniProgram({ fail: () => wx.showToast({ title: '请用手势关闭', icon: 'none' }) }) }
    })
  }
})
