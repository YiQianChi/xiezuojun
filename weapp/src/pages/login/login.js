// 壳登录页：微信一键登录 + 邮箱登录；登录成功 token 落 storage 后跳 index（web-view）
Page({
  data: {
    email: '',
    password: '',
    loading: false,
    wxLoading: false
  },
  onEmail(e) { this.setData({ email: e.detail.value }) },
  onPwd(e) { this.setData({ password: e.detail.value }) },

  saveTokens(d) {
    if (d.access_token) wx.setStorageSync('xzj_token', d.access_token)
    if (d.refresh_token) wx.setStorageSync('xzj_rt', d.refresh_token)
    wx.redirectTo({ url: '/pages/index/index' })
  },

  // 微信一键登录：wx.login 拿 code → 后端换 openid 建号/登录
  async onWxLogin() {
    if (this.data.wxLoading) return
    this.setData({ wxLoading: true })
    try {
      const loginRes = await new Promise((resolve, reject) => {
        wx.login({ success: resolve, fail: reject })
      })
      if (!loginRes.code) throw new Error('no code')
      const res = await new Promise((resolve, reject) => {
        wx.request({
          url: 'https://xiezuojun.com/api/auth/wx-login',
          method: 'POST',
          data: { code: loginRes.code },
          header: { 'Content-Type': 'application/json' },
          success: resolve,
          fail: reject
        })
      })
      const d = res.data && res.data.data
      if (res.statusCode === 200 && d && d.access_token) {
        this.saveTokens(d)
        return
      }
      wx.showToast({ title: (res.data && res.data.error) || '微信登录失败，试试邮箱登录', icon: 'none' })
    } catch (e) {
      wx.showToast({ title: '网络错误，请重试', icon: 'none' })
    }
    this.setData({ wxLoading: false })
  },

  async onLogin() {
    const { email, password } = this.data
    if (!email || !password) {
      wx.showToast({ title: '请填写邮箱和密码', icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const res = await new Promise((resolve, reject) => {
        wx.request({
          url: 'https://xiezuojun.com/api/auth/login',
          method: 'POST',
          data: { email, password },
          header: { 'Content-Type': 'application/json' },
          success: resolve,
          fail: reject
        })
      })
      const d = res.data && res.data.data
      if (res.statusCode === 200 && d && d.access_token) {
        this.saveTokens(d)
        return
      }
      wx.showToast({ title: (res.data && res.data.error) || '邮箱或密码错误', icon: 'none' })
    } catch (e) {
      wx.showToast({ title: '网络错误，请重试', icon: 'none' })
    }
    this.setData({ loading: false })
  },

  goRegister() {
    // 注册/找回密码去网页版处理。注：不用剪贴板（隐私接口，需隐私指引+授权弹窗，壳保持零隐私接口）
    wx.showModal({
      title: '注册账号',
      content: '请在电脑或手机浏览器访问 xiezuojun.com 注册，注册后回来登录。',
      showCancel: false
    })
  }
})
