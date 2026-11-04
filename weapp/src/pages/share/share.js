// 章节分享页：由网页「转发本章」按钮拉起（navigateTo），参数带在页面路径里——
// 不依赖 web-view postMessage 的送达时机，转发卡片 100% 是本章
const app = getApp()

Page({
  data: { code: '', title: '', book: '' },
  onLoad(options) {
    this.setData({
      code: options.code || '',
      title: decodeURIComponent(options.title || ''),
      book: decodeURIComponent(options.book || '')
    })
  },
  onShareAppMessage() {
    return {
      title: '《' + (this.data.book || '作品') + '》' + this.data.title + ' | 写作菌',
      path: '/pages/index/index?web=read/' + this.data.code,
      imageUrl: '/images/share-cover.jpg' // 转发卡片缩略图（微信按 5:4 裁切）
    }
  },
  goBack() {
    wx.navigateBack()
  }
})
