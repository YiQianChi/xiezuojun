const app = getApp()

Page({
  data: {
    url: '',
    loadFailed: false
  },
  onLoad() {
    this.setData({ url: app.globalData.site })
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
