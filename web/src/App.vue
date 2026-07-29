<template>
  <nav class="top-nav" v-if="showNav">
    <div class="nav-left">
      <router-link to="/" class="logo">写作菌</router-link>
    </div>
    <div class="nav-right">
      <a class="nav-link qq-feedback" href="https://qm.qq.com/q/1091462449" target="_blank" rel="noopener">QQ问题反馈群</a>
      <router-link to="/bookshelf" class="nav-link">书架</router-link>
      <button v-if="isLoggedIn" @click="logout" class="nav-link logout">退出</button>
      <router-link v-else to="/login" class="nav-link">登录</router-link>
    </div>
  </nav>
  <main class="app-main" :class="{ 'with-nav': showNav, 'editor-mode': route.name === 'Editor' }">
    <router-view />
  </main>
  <!-- Toast 容器 -->
  <div class="toast-container">
    <Transition-group name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast-item" :class="t.type">
        {{ t.message }}
      </div>
    </Transition-group>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toasts } from './composables/useToast'

const route = useRoute()
const router = useRouter()
const showNav = computed(() => route.name !== 'Editor' && route.name !== 'Login' && route.name !== 'Register')

const isLoggedIn = ref(!!localStorage.getItem('access_token'))

// 全局禁用浏览器原生右键菜单（应用内有自定义右键交互）
onMounted(() => {
  document.addEventListener('contextmenu', (e) => e.preventDefault())
})

function logout() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  isLoggedIn.value = false
  router.push('/login')
}
</script>

<style scoped>
.top-nav {
  flex-shrink: 0;
  height: var(--nav-height);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 32px;
  background: var(--glass-bg-solid);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
  z-index: 100;
}

.app-main {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
}

.app-main.editor-mode {
  overflow: hidden;
}

.logo {
  font-family: var(--font-serif);
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
  text-decoration: none;
  letter-spacing: 1px;
}

.nav-right {
  display: flex;
  gap: 24px;
  align-items: center;
}

.nav-link {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  background: none;
  border: none;
  cursor: pointer;
  transition: color var(--transition-fast);
  padding: 0;
}

.nav-link:hover {
  color: var(--text-primary);
}

.logout {
  font-family: inherit;
}

/* Toast */
.toast-container {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.toast-item {
  padding: 10px 20px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #fff;
  backdrop-filter: blur(20px);
  box-shadow: 0 4px 16px rgba(0,0,0,.15);
  pointer-events: auto;
}

.toast-item.error {
  background: rgba(255,59,48,.92);
}

.toast-item.success {
  background: rgba(52,199,89,.92);
}

.toast-item.info {
  background: rgba(0,122,255,.92);
}

.toast-enter-active, .toast-leave-active {
  transition: all .3s ease;
}
.toast-enter-from, .toast-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}
</style>