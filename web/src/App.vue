<template>
  <nav class="top-nav" v-if="showNav">
    <div class="nav-left">
      <router-link to="/" class="logo">写作菌</router-link>
    </div>
    <div class="nav-right">
      <router-link to="/bookshelf" class="nav-link">书架</router-link>
      <button v-if="isLoggedIn" @click="logout" class="nav-link logout">退出</button>
      <router-link v-else to="/login" class="nav-link">登录</router-link>
    </div>
  </nav>
  <main class="app-main" :class="{ 'with-nav': showNav, 'editor-mode': route.name === 'Editor' }">
    <router-view />
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const showNav = computed(() => route.name !== 'Editor' && route.name !== 'Login' && route.name !== 'Register')

const isLoggedIn = ref(!!localStorage.getItem('access_token'))

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
</style>