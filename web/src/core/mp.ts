// 小程序 web-view ↔ 壳的桥接工具
export const isMiniProgramEnv =
  /miniProgram/i.test(navigator.userAgent) || (window as any).__wxjs_environment === 'miniProgram'

// 跳壳的页面（jweixin 没加载时动态补载；redirectTo 替换掉承载 web-view 的 index 页）
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
