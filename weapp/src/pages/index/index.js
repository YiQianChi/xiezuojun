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
    return app.globalData.site + '?xzj_at=' + encodeURIComponent(at) + (rt ? '&xzj_rt=' + encodeURIComponent(rt) : '')
  },
  async onLoad() {
    const at = wx.getStorageSync('xzj_token')
    const rt = wx.getStorageSync('xzj_rt')
    if (!at && !rt) {
      wx.redirectTo({ url: '/pages/login/login' })
      return
    }
    if (at) {
      // 桥接 token 进 web-view（一次性，storage 里的留作续期用）
      this.setData({ url: this.buildUrl(at, rt) })
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
  }
})
