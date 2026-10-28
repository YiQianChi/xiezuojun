const app = getApp()

Page({
  data: {
    url: '',
    loadFailed: false
  },
  onLoad() {
    this.setData({ url: app.globalData.site })
  },
  onShow() {
    // 每次回到本页都检查壳侧登录态，把 token 桥接进 web-view（小程序与网页 localStorage 隔离，走 URL 传递）
    const at = wx.getStorageSync('xzj_token')
    if (!at) return
    wx.removeStorageSync('xzj_token') // 一次性，防重复拼接
    const rt = wx.getStorageSync('xzj_rt') || ''
    const sep = this.data.url.includes('?') ? '&' : '?'
    this.setData({ url: this.data.url + sep + 'xzj_at=' + encodeURIComponent(at) + (rt ? '&xzj_rt=' + encodeURIComponent(rt) : '') })
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
