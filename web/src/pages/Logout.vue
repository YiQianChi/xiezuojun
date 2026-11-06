<template>
  <div class="lo">
    <p class="lo-tip">{{ done ? '已退出登录' : '正在退出…' }}</p>
    <p class="lo-sub">可以游客身份继续浏览</p>
    <button class="lo-btn" @click="goHome">回到首页</button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

// 小程序原生「退出登录」跳到这里清 webview localStorage 的 access_token——
// 原生存储和 webview 存储是两个世界，只清原生会出现「退出了书架还有书」
const done = ref(false)
const router = useRouter()
onMounted(() => {
  try {
    localStorage.removeItem('access_token')
    localStorage.removeItem('xzj-remember')
  } catch { /* ignore */ }
  done.value = true
})
function goHome() { router.push('/') }
</script>

<style scoped>
.lo { min-height: 60vh; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px; }
.lo-tip { font-size: 17px; font-weight: 600; color: var(--text-primary); }
.lo-sub { font-size: 13px; color: var(--text-secondary); margin-bottom: 18px; }
.lo-btn { padding: 10px 34px; border-radius: 999px; border: none; background: var(--accent); color: #fff; font-size: 15px; cursor: pointer; }
</style>
