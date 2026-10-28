Page({
  data: {
    email: '',
    password: '',
    loading: false
  },
  onEmail(e) { this.setData({ email: e.detail.value }) },
  onPwd(e) { this.setData({ password: e.detail.value }) },
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
      if (res.statusCode === 200 && res.data && res.data.token) {
        wx.setStorageSync('xzj_token', res.data.token)
        wx.setStorageSync('xzj_email', email)
        wx.redirectTo({ url: '/pages/index/index' })
      } else {
        wx.showToast({ title: (res.data && res.data.error) || '邮箱或密码错误', icon: 'none' })
      }
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
