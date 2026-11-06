// 原生公开阅读页：分享卡片落地（免登录），文末转化卡
const app = getApp()

function htmlToText(html) {
  if (!html) return []
  return html
    .replace(/<\/(p|div|h[1-6]|li)>/gi, '\n')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"')
    .split('\n').map(l => l.trim()).filter(Boolean)
}

Page({
  data: {
    loading: true,
    none: false,
    book: '',
    title: '',
    wc: 0,
    paras: []
  },

  onLoad(options) {
    const code = options.code || ''
    this._code = code
    wx.request({
      url: app.globalData.site + '/api/public/read/' + code,
      method: 'GET',
      success: res => {
        const d = res.data && res.data.data
        if (res.statusCode === 200 && d) {
          this.setData({ loading: false, book: d.project_name || '', title: d.title || '', wc: d.word_count || 0, paras: htmlToText(d.content) })
          wx.setNavigationBarTitle({ title: (d.title || '阅读') + ' · 写作菌' })
        } else {
          this.setData({ loading: false, none: true })
        }
      },
      fail: () => this.setData({ loading: false, none: true })
    })
  },

  goCreate() {
    if (wx.getStorageSync('xzj_token')) {
      wx.reLaunch({ url: '/pages/index/index' }) // 原生页已全撤：回 index 网页壳
    } else {
      wx.reLaunch({ url: '/pages/index/index' }) // 游客营销页
    }
  },

  onShareAppMessage() {
    return {
      title: '《' + (this.data.book || '作品') + '》' + this.data.title + ' | 写作菌',
      path: '/pages/read/read?code=' + (this._code || ''),
      imageUrl: 'https://xiezuojun.com/share-card.jpg'
    }
  }
})
