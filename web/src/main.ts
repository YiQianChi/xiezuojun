import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './style.css'

/* 主题引导：xzj-theme = xuan(米宣默认)/qing/wu/yangpi/mo(冷墨)/ye(夜灯)；旧 theme=dark 映射 mo */
;(() => {
  let t = localStorage.getItem('xzj-theme')
  if (!t) t = localStorage.getItem('theme') === 'dark' ? 'mo' : 'xuan'
  const el = document.documentElement
  el.classList.remove('dark', 'theme-ye', 'theme-qing', 'theme-wu', 'theme-yangpi')
  if (t === 'mo') el.classList.add('dark')
  else if (t === 'ye') el.classList.add('dark', 'theme-ye')
  else if (['qing', 'wu', 'yangpi'].includes(t)) el.classList.add('theme-' + t)
})()

const app = createApp(App)

app.use(router)

app.mount('#app')
