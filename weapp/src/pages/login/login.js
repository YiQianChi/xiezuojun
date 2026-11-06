// 壳登录页：微信一键登录 + 邮箱登录；登录成功 token 落 storage 后跳 index（web-view）
Page({
  onLoad() {
    // 进登录页=旧登录态作废：清壳侧 token（网页里退出后会 redirectTo 到这里；不清的话
    // 点左上角主页 index 又拿旧 token 自动登回来）
    wx.removeStorageSync('xzj_token')
    wx.removeStorageSync('xzj_rt')
  },

  data: {
    email: '',
    password: '',
    loading: false,
    wxLoading: false,
    agreed: false // 微信审核：默认不勾选协议，用户自主勾选后才可登录
  },
  onEmail(e) { this.setData({ email: e.detail.value }) },
  onPwd(e) { this.setData({ password: e.detail.value }) },
  toggleAgree() { this.setData({ agreed: !this.data.agreed }) },
  needAgree() { wx.showToast({ title: '请先阅读并同意协议', icon: 'none' }) },

  saveTokens(d) {
    if (d.access_token) wx.setStorageSync('xzj_token', d.access_token)
    if (d.refresh_token) wx.setStorageSync('xzj_rt', d.refresh_token)
    // 登录成功 → index（web-view 桥接进网页工作台；原生页已全撤）
    wx.reLaunch({ url: '/pages/index/index' })
  },

  // 微信一键登录：getPhoneNumber 授权 + wx.login 一次完成；用户拒绝手机号授权则降级纯微信登录
  async onWxLogin(e) {
    if (this.data.wxLoading) return
    if (!this.data.agreed) { this.needAgree(); return } // 双保险：按钮已条件渲染，此处防直接调用
    this.setData({ wxLoading: true })
    try {
      const loginRes = await new Promise((resolve, reject) => {
        wx.login({ success: resolve, fail: reject })
      })
      if (!loginRes.code) throw new Error('no code')
      const phoneCode = e.detail && e.detail.code ? e.detail.code : ''
      if (!phoneCode && e.detail && e.detail.errMsg && e.detail.errMsg.indexOf('deny') < 0) {
        wx.showToast({ title: '手机号: ' + e.detail.errMsg, icon: 'none', duration: 3000 })
      }
      const res = await new Promise((resolve, reject) => {
        wx.request({
          url: phoneCode
            ? 'https://xiezuojun.com/api/auth/wx-phone-login'
            : 'https://xiezuojun.com/api/auth/wx-login',
          method: 'POST',
          data: phoneCode ? { code: loginRes.code, phone_code: phoneCode } : { code: loginRes.code },
          header: { 'Content-Type': 'application/json' },
          success: resolve,
          fail: reject
        })
      })
      const d = res.data && res.data.data
      if (res.statusCode === 200 && d && d.access_token) {
        if (d.merged) wx.showToast({ title: '已并入你的老账号', icon: 'none' })
        this.saveTokens(d)
        return
      }
      wx.showToast({ title: (res.data && res.data.error) || '手机号登录失败，试试邮箱登录', icon: 'none' })
    } catch (e) {
      wx.showToast({ title: '网络错误，请重试', icon: 'none' })
    }
    this.setData({ wxLoading: false })
  },

  async onLogin() {
    const { email, password } = this.data
    if (!this.data.agreed) { this.needAgree(); return }
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

  openAgreement() { wx.navigateTo({ url: '/pages/index/index?web=agreement' }) },
  openPrivacy() { wx.navigateTo({ url: '/pages/index/index?web=privacy' }) },
  goRegister() {
    // 注册页在 web-view 里跑（邮箱+验证码+密码），注册成功自动发登录态进书架
    wx.navigateTo({ url: '/pages/index/index?web=register' })
  }
})
