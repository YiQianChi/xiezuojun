<template>
  <nav class="top-nav" v-if="showNav">
    <div class="nav-left">
      <router-link to="/" class="logo">
        <img src="/icons/icon-192.png" alt="写作菌" class="logo-icon" />
        写作菌
      </router-link>
    </div>
    <div class="nav-right">
      <button v-if="canInstall" @click="installApp" class="nav-link install-link">⬇ 安装到桌面</button>
      <button class="nav-link" @click="showQQModal = true">QQ问题反馈群</button>
      <router-link to="/bookshelf" class="nav-link">书架</router-link>
      <button v-if="isLoggedIn" class="nav-link profile-link" title="个人设置" @click="profileOpen = true">
        <span class="nav-avatar">{{ profileAvatar || (profileNickname || '我').slice(0, 1) }}</span>
        <span class="nav-nick">{{ profileNickname || '我的' }}</span>
      </button>
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
  <!-- QQ群弹窗 -->
  <Teleport to="body">
    <div v-if="showQQModal" class="qq-overlay" @click="showQQModal = false">
      <div class="qq-modal" @click.stop>
        <div class="qq-head">
          <span>💬 问题反馈群</span>
          <button @click="showQQModal = false" class="qq-close">✕</button>
        </div>
        <div class="qq-body">
          <p class="qq-desc">遇到问题或建议？加入QQ群反馈：</p>
          <div class="qq-number">1091462449</div>
          <button class="qq-copy-btn" @click="copyQQ">复制群号</button>
        </div>
      </div>
    </div>
  </Teleport>
  <ProfileModal v-model:open="profileOpen" />
  <OnboardingTour ref="onboardingRef" />
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toasts, toast } from './composables/useToast'
import { usePwaInstall } from './composables/usePwaInstall'
import OnboardingTour from './components/OnboardingTour.vue'
import ProfileModal from './components/ProfileModal.vue'

const route = useRoute()
const router = useRouter()
const showNav = computed(() => route.name !== 'Editor' && route.name !== 'Login' && route.name !== 'Register')

const isLoggedIn = ref(false)
// 导航栏头像/昵称（登录后拉取；个人设置页改完派发 xzj-profile-updated 事件刷新）
const profileOpen = ref(false)
const profileNickname = ref('')
const profileAvatar = ref('')
async function refreshProfile() {
  profileNickname.value = ''
  profileAvatar.value = ''
  const token = localStorage.getItem('access_token')
  if (!token) return
  // 缓存优先（立即显示），再静默拉最新
  try {
    profileNickname.value = localStorage.getItem('xzj-nickname') || ''
    profileAvatar.value = localStorage.getItem('xzj-avatar') || ''
  } catch {}
  try {
    const { userApi } = await import('./core/api')
    const res = await userApi.me()
    const u = res.data?.data
    if (u) {
      profileNickname.value = u.nickname || ''
      profileAvatar.value = u.avatar || ''
      try {
        localStorage.setItem('xzj-nickname', profileNickname.value)
        localStorage.setItem('xzj-avatar', profileAvatar.value)
      } catch {}
    }
  } catch {}
}
window.addEventListener('xzj-profile-updated', refreshProfile)
// 登录/登出后路由会跳转，每次导航都重读一次 token，保证导航栏状态实时
function refreshLoginState() {
  const was = isLoggedIn.value
  isLoggedIn.value = !!localStorage.getItem('access_token')
  if (isLoggedIn.value && !was) refreshProfile()
}
watch(() => route.fullPath, refreshLoginState)
const showQQModal = ref(false)
const onboardingRef = ref<{ start: () => void; forceStart: () => void } | null>(null)
const { canInstall, promptInstall } = usePwaInstall()

async function installApp() {
  const outcome = await promptInstall()
  if (outcome === 'accepted') toast.success('已安装！桌面上的「写作菌」双击即用')
  else if (outcome === 'unavailable') {
    // 浏览器没抛 beforeinstallprompt（已安装/不支持/已 standalone）——给手动指引
    toast.info('请在浏览器地址栏右侧 ⋯ 菜单里选「安装应用」')
  }
}

function copyQQ() {
  navigator.clipboard?.writeText('1091462449').then(() => {
    // ponytail: 简单提示，复用 toast
  }).catch(() => {})
}

function onReplayOnboarding() {
  onboardingRef.value?.forceStart()
}
onMounted(() => {
  refreshLoginState()
  if (isLoggedIn.value) refreshProfile()
  document.addEventListener('contextmenu', (e) => e.preventDefault())
  window.addEventListener('replay-onboarding', onReplayOnboarding)
})

function logout() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  try { localStorage.removeItem('xzj-nickname'); localStorage.removeItem('xzj-avatar') } catch {}
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
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.logo-icon {
  width: 26px;
  height: 26px;
  border-radius: 7px;
  display: block;
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

.profile-link { display: inline-flex; align-items: center; gap: 6px; }
.nav-avatar {
  width: 22px; height: 22px; border-radius: 50%; background: var(--accent, #007aff);
  color: #fff; font-size: 13px; line-height: 22px; text-align: center; overflow: hidden; flex-shrink: 0;
}
.nav-nick { max-width: 6em; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.logout {
  font-family: inherit;
}

.install-link {
  color: var(--accent, #007aff);
  font-weight: 600;
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

/* QQ群弹窗 */
.qq-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}
.qq-modal {
  background: var(--bg-secondary);
  border-radius: 14px;
  width: 320px;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}
.qq-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}
.qq-close {
  background: none;
  border: none;
  font-size: 18px;
  color: var(--text-tertiary);
  cursor: pointer;
}
.qq-body {
  padding: 24px 20px;
  text-align: center;
}
.qq-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 12px;
}
.qq-number {
  font-size: 28px;
  font-weight: 700;
  color: var(--accent, #007aff);
  letter-spacing: 2px;
  margin-bottom: 16px;
  font-family: var(--font-sans);
}
.qq-copy-btn {
  padding: 8px 24px;
  border-radius: 8px;
  border: none;
  background: var(--accent, #007aff);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity .15s ease;
}
.qq-copy-btn:hover { opacity: .85; }
</style>