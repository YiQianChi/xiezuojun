// 小程序 web-view ↔ 壳的桥接工具
export const isMiniProgramEnv =
  /miniProgram/i.test(navigator.userAgent) || (window as any).__wxjs_environment === 'miniProgram'

// 跳壳的页面（jweixin 没加载时动态补载）
export function mpNavigate(url: string) {
  const w = window as any
  const go = () => w.wx?.miniProgram?.redirectTo?.({ url })
  if (w.wx?.miniProgram) {
    go()
    return
  }
  const s = document.createElement('script')
  s.src = 'https://res.wx.qq.com/open/js/jweixin-1.6.0.js'
  s.onload = go
  document.body.appendChild(s)
}

// 在壳里新开一层页面（章节分享页用：参数带在路径里，转发卡片零竞态）
export function mpPush(url: string) {
  const w = window as any
  const go = () => w.wx?.miniProgram?.navigateTo?.({ url })
  if (w.wx?.miniProgram) {
    go()
    return
  }
  const s = document.createElement('script')
  s.src = 'https://res.wx.qq.com/open/js/jweixin-1.6.0.js'
  s.onload = go
  document.body.appendChild(s)
}
