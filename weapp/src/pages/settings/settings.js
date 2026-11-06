// 原生设置页：改密码 / 退出登录 / 协议隐私 / 版本
const api = require('../../utils/api')

Page({
  data: {
    version: '0.5.5',
    email: '',
    showPwd: false,
    pwd1: '',
    pwd2: '',
    saving: false
  },

  onShow() {
    try {
      const acc = wx.getAccountInfoSync()
      const v = acc.miniProgram.version || this.data.version
      this.setData({ version: v })
    } catch (e) {}
    api.get('/user/me').then(res => {
      const me = res.data || res
      this.setData({ email: me.email || '' })
    }).catch(() => {})
  },

  openPwd() { this.setData({ showPwd: !this.data.showPwd, pwd1: '', pwd2: '' }) },
  onPwd1(e) { this.setData({ pwd1: e.detail.value }) },
  onPwd2(e) { this.setData({ pwd2: e.detail.value }) },

  async savePwd() {
    const { pwd1, pwd2, saving } = this.data
    if (saving) return
    if (pwd1.length < 6) { wx.showToast({ title: '密码至少 6 位', icon: 'none' }); return }
    if (pwd1 !== pwd2) { wx.showToast({ title: '两次输入不一致', icon: 'none' }); return }
    this.setData({ saving: true })
    try {
      await api.post('/user/change-password', { new_password: pwd1 })
      wx.showToast({ title: '已保存' })
      this.setData({ showPwd: false, saving: false })
    } catch (e) {
      this.setData({ saving: false })
      wx.showToast({ title: (e.data && e.data.error) || '保存失败', icon: 'none' })
    }
  },

  logout() {
    wx.showModal({
      title: '退出登录？',
      confirmText: '退出',
      confirmColor: '#d05252',
      success: r => {
        if (!r.confirm) return
        wx.removeStorageSync('xzj_token')
        wx.removeStorageSync('xzj_rt')
        // 走 webview 的 /logout 清网页侧 localStorage（和原生存储是两个世界，
        // 只清原生会出现「退出了、书架还有书」）；顺带落回游客营销页
        wx.reLaunch({ url: '/pages/index/index?web=logout' })
      }
    })
  },

  openAgreement() { wx.navigateTo({ url: '/pages/index/index?web=agreement' }) },
  openPrivacy() { wx.navigateTo({ url: '/pages/index/index?web=privacy' }) }
})
