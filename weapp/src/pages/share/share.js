// 分享页：由网页按钮拉起（navigateTo），参数带在页面路径里——
// 不依赖 web-view postMessage 的送达时机，转发卡片 100% 是当前内容
// kind=chapter 章节公开读（默认）/ kind=sprint 拼字房邀请
const app = getApp()

Page({
  data: { code: '', title: '', book: '', kind: '', label: '', pw: '' },
  onLoad(options) {
    this.setData({
      kind: options.kind || '',
      code: options.code || '',
      pw: options.pw ? decodeURIComponent(options.pw) : '',
      title: decodeURIComponent(options.title || ''),
      book: decodeURIComponent(options.book || ''),
      label: decodeURIComponent(options.label || '')
    })
  },
  onShareAppMessage() {
    // 拼字邀请卡：走壳已有 ?web= 深链——好友点开 → web-view 落 /bookshelf?sprint=码，
    // 网页侧已做闭环（未登录走登录回跳、已登录自动进书弹「加入房间」）
    if (this.data.kind === 'sprint') {
      let deep = 'bookshelf?sprint=' + this.data.code
      if (this.data.pw) deep += '&pw=' + encodeURIComponent(this.data.pw)
      return {
        title: '一起拼字！' + (this.data.label ? this.data.label + ' · ' : '') + '房间 ' + this.data.code + ' | 写作菌',
        path: '/pages/index/index?web=' + encodeURIComponent(deep),
        imageUrl: 'https://xiezuojun.com/share-card.jpg' // 网络图（包内路径不生效），已预裁 5:4
      }
    }
    return {
      title: '《' + (this.data.book || '作品') + '》' + this.data.title + ' | 写作菌',
      path: '/pages/read/read?code=' + this.data.code,
      imageUrl: 'https://xiezuojun.com/share-card.jpg' // 网络图（包内路径不生效），已预裁 5:4
    }
  },
  goBack() {
    wx.navigateBack()
  }
})
