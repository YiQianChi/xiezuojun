const app = getApp()

// 首页=web-view 承载线上站。进入前检查壳侧登录态：
// - 无 token → 跳原生登录页
// - 有 access token → 桥接进 web-view（一次拼 URL，token 从 storage 清掉防重复）
// - access 过期但有 refresh → 静默换新再进（30 天内免登录）
Page({
  data: {
    url: '',
    loadFailed: false
  },
  buildUrl(at, rt) {
    // 首页工作台（Boss 拍板方案C：数据条+快捷宫格+最近打开）；登录后进这里，不再是直接弹书架
    // 注：web-view 页原生导航栏无法去除（官方文档：6.7.2 起 navigationStyle:custom 对 web-view 无效），
    // 全屏方案改走「小程序内隐藏网页顶栏」——见 web App.vue showNav
    return app.globalData.site + '/home?xzj_at=' + encodeURIComponent(at) + (rt ? '&xzj_rt=' + encodeURIComponent(rt) : '')
  },
  async onLoad(options) {
    // 深链：?web=editor/PUBID&xzj_at=…（从原生首页/书架打开编辑器、导入面板——token 桥接拼进 URL）
    if (options && options.web) {
      let path = decodeURIComponent(options.web)
      if (options.xzj_at) {
        path += (path.indexOf('?') >= 0 ? '&' : '?') + 'xzj_at=' + encodeURIComponent(options.xzj_at)
        if (options.xzj_rt) path += '&xzj_rt=' + encodeURIComponent(options.xzj_rt)
      }
      this.setData({ url: app.globalData.site + '/' + path })
      return
    }
    const at = wx.getStorageSync('xzj_token')
    const rt = wx.getStorageSync('xzj_rt')
    if (!at && !rt) {
      // 微信审核要求：先浏览后自主授权登录——无登录态进营销首页游客浏览，登录入口在网页里跳回壳登录页
      this.setData({ url: app.globalData.site + '/' })
      return
    }
    if (at) {
      // 兜底：老入口直接进 index 且带登录态 → 原生首页（正常流程 home 是启动页，轮不到这里）
      wx.reLaunch({ url: '/pages/home/home' })
      return
    }
    // 无 access 有 refresh：静默续期
    const refreshed = await this.tryRefresh(rt)
    if (refreshed) return
    this.clearAndLogin()
  },
  async tryRefresh(rt) {
    try {
      const res = await new Promise((resolve, reject) => {
        wx.request({
          url: 'https://xiezuojun.com/api/auth/refresh',
          method: 'POST',
          data: { refresh_token: rt },
          header: { 'Content-Type': 'application/json' },
          success: resolve,
          fail: reject
        })
      })
      const d = res.data && res.data.data
      if (res.statusCode === 200 && d && d.access_token) {
        wx.setStorageSync('xzj_token', d.access_token)
        if (d.refresh_token) wx.setStorageSync('xzj_rt', d.refresh_token)
        this.setData({ url: this.buildUrl(d.access_token, d.refresh_token) })
        return true
      }
    } catch (e) { /* 网络问题落到重登 */ }
    return false
  },
  clearAndLogin() {
    wx.removeStorageSync('xzj_token')
    wx.removeStorageSync('xzj_rt')
    wx.redirectTo({ url: '/pages/login/login' })
  },
  onError(e) {
    console.error('web-view error', e.detail)
    this.setData({ loadFailed: true })
  },
  onRetry() {
    this.setData({ loadFailed: false, url: '' })
    setTimeout(() => this.setData({ url: app.globalData.site }), 200)
  },

  // ===== 分享 =====
  // 网页里的章节分享会 postMessage 过来。web-view 的消息只在特定时机（转发/后退/销毁）送达，
  // 有竞态——所以收到就写进 storage，转发时读内存+storage 双保险，没拿到才落默认卡片
  onWebViewMsg(e) {
    const list = (e.detail && e.detail.data) || []
    for (let i = list.length - 1; i >= 0; i--) {
      if (list[i] && list[i].type === 'xzj-share') {
        this.shareMsg = list[i]
        try { wx.setStorageSync('xzj_share', list[i]) } catch (err) {}
        return
      }
    }
  },
  onShareAppMessage() {
    let s = this.shareMsg || wx.getStorageSync('xzj_share')
    // storage 里的旧分享 10 分钟后失效，避免用户想默认转发时还带着老章节
    if (s && s.ts && Date.now() - s.ts > 10 * 60 * 1000) s = null
    if (s && s.code) {
      return {
        title: '《' + (s.book || '作品') + '》' + (s.title || '') + ' | 写作菌',
        path: '/pages/index/index?web=read/' + s.code,
        imageUrl: 'https://xiezuojun.com/share-card.jpg' // 包内路径不生效，用网络图
      }
    }
    return {
      title: '写作菌 · 让灵感不被打扰',
      path: '/pages/index/index',
      imageUrl: 'https://xiezuojun.com/share-card.jpg'
    }
  },
  // 分享到朋友圈（仅 Android 微信支持，单页模式打开）
  onShareTimeline() {
    return {
      title: '写作菌 · 让灵感不被打扰',
      query: '',
      imageUrl: 'https://xiezuojun.com/share-card.jpg' // 朋友圈卡片只认网络图
    }
  }
})
