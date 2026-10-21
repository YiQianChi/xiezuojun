import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [vue()],
  // ponytail-harmony: 鸿蒙壳从 resource://rawfile/ 加载，绝对路径 /assets 会解析失败（白屏）。
  // harmony 模式用相对路径 + hash 路由（router/index.ts 同步判断）。
  base: mode === 'harmony' ? './' : '/',
}))
