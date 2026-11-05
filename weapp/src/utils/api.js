// 壳内 API 工具：带 token 的 wx.request 封装
const app = getApp()

function token() { return wx.getStorageSync('xzj_token') || '' }

function call(method, path, data) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: app.globalData.site + '/api' + path,
      method,
      data: data || {},
      header: {
        'Content-Type': 'application/json',
        'Authorization': token() ? 'Bearer ' + token() : ''
      },
      success: res => {
        if (res.statusCode === 401) {
          // 登录态失效 → 回登录页
          wx.removeStorageSync('xzj_token')
          wx.removeStorageSync('xzj_rt')
          wx.reLaunch({ url: '/pages/login/login' })
          reject(new Error('unauthorized'))
          return
        }
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data)
        else reject({ status: res.statusCode, data: res.data })
      },
      fail: reject
    })
  })
}

module.exports = {
  get: p => call('GET', p),
  post: (p, d) => call('POST', p, d),
  patch: (p, d) => call('PATCH', p, d),
  del: p => call('DELETE', p),
  // 打开编辑器（web-view）：桥接 token 进 URL
  openEditor(publicId) {
    const at = wx.getStorageSync('xzj_token')
    const rt = wx.getStorageSync('xzj_rt')
    let url = '/pages/index/index?web=editor/' + publicId
    if (at) url += '&xzj_at=' + encodeURIComponent(at)
    if (rt) url += '&xzj_rt=' + encodeURIComponent(rt)
    wx.navigateTo({ url })
  },
  coverUrl: u => (u && u.indexOf('/') === 0) ? app.globalData.site + u : (u || '')
}
