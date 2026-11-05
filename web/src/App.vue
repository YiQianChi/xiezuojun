<template>
  <nav class="top-nav" v-if="showNav">
    <div class="nav-left">
      <router-link to="/" class="logo">
        <img src="/icons/icon-192.png" alt="写作菌" class="logo-icon" />
        写作菌
      </router-link>
    </div>
    <div class="nav-right">
      <button v-if="canInstall && !isMiniProgram" @click="installApp" class="nav-link install-link">⬇ 安装到桌面</button>
      <button v-if="!isMiniProgram" class="nav-link qq-link" data-label="反馈" @click="showQQModal = true">QQ问题反馈群</button>
      <router-link to="/bookshelf" class="nav-link">书架</router-link>
      <button v-if="isLoggedIn" ref="profileBtnRef" class="nav-link profile-link" title="账号菜单" @click="profileRef?.openMenu($event.currentTarget as HTMLElement)">
        <span class="nav-avatar">{{ profileAvatar || (profileNickname || '我').slice(0, 1) }}</span>
        <span class="nav-nick">{{ profileNickname || '我的' }}</span>
        <span class="nav-caret">▾</span>
      </button>
      <router-link v-else to="/login" class="nav-link">登录</router-link>
    </div>
  </nav>
    <!-- 小程序 web-view：网页顶栏已隐藏，返回/主页子栏；编辑器层由原生栏自带官方返回箭头，无需网页侧按钮 -->
    <div v-if="showMpNavBtns" class="mp-nav-btns">
      <button class="mp-nav-btn" title="返回" @click="goBackMp">
        <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
      </button>
      <button class="mp-nav-btn" title="主页" @click="goHomeMp">
        <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 10.5L12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/></svg>
      </button>
    </div>
  <main class="app-main" :class="{ 'with-nav': showNav, 'editor-mode': route.name === 'Editor', 'mp-pad': showMpNavBtns }">
    <router-view />
  </main>
  <!-- Toast 容器 -->
  <div class="toast-container">
    <Transition-group name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast-item" :class="t.type">
        <span>{{ t.message }}</span>
        <button v-if="t.action" class="toast-action" @click="t.action!.onClick(); dismissToast(t.id)">{{ t.action.label }}</button>
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
  <ProfileModal ref="profileRef" />
  <OnboardingTour ref="onboardingRef" />
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toasts, toast, dismissToast } from './composables/useToast'
import { usePwaInstall } from './composables/usePwaInstall'
import OnboardingTour from './components/OnboardingTour.vue'
import ProfileModal from './components/ProfileModal.vue'
import { isMiniProgramEnv, mpBack, mpRelaunch } from './core/mp'

const route = useRoute()

// 小程序 web-view：原生导航栏去不掉（官方限制），改为隐藏网页自己的顶栏只留一条栏；
// --mp-safe-top/--mp-capsule-pad CSS 变量留作日后开放 custom 导航时用（现在恒为 0）
const inMiniProgram = isMiniProgramEnv
const router = useRouter()
const showNav = computed(() => !inMiniProgram && route.name !== 'Editor' && route.name !== 'Login' && route.name !== 'Register')

// 小程序顶部子栏：营销首页('/')是壳根页没有可返回层级隐藏；编辑器层的原生栏自带官方返回箭头（navigateTo 自动出现），网页侧不再重复给
const showMpNavBtns = computed(() => inMiniProgram && route.path !== '/' && route.name !== 'Login' && route.name !== 'Register' && route.name !== 'Editor')
function goBackMp() {
  // 有 SPA 内部上一页（router 记录在 history.state.back）才走网页后退，否则=关掉这层 web-view 回原生页
  const st: any = window.history.state
  if (st?.back) { window.history.back(); return }
  mpBack()
}
function goHomeMp() {
  // 小程序里「主页」= 回原生工作台；未登录回营销首页
  if (isLoggedIn.value) { mpRelaunch('/pages/home/home'); return }
  router.push('/')
}

const isLoggedIn = ref(false)
// 导航栏头像/昵称（登录后拉取；个人设置页改完派发 xzj-profile-updated 事件刷新）
const profileRef = ref<{ openMenu: (a: HTMLElement) => void; openForPenName: () => void; openSecurityEmail: () => void; openPwd: () => void } | null>(null)
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
      // 昵称是邮箱/空 → 弹「起个笔名」（搭子配对只展示笔名，保护邮箱）
      if (u.needs_pen_name && !sessionStorage.getItem('xzj-penname-asked')) {
        try { sessionStorage.setItem('xzj-penname-asked', '1') } catch {}
        profileRef.value?.openForPenName()
      }
      // 没密码的用户（微信/扫码/手机号建号）：上线提醒设置密码（手机号用户可直接手机号+密码登录）
      const s = u.security || {}
      if (!s.has_password && !localStorage.getItem('xzj_skip_setpwd')
        && route.path !== '/home' && !sessionStorage.getItem('xzj-pwd-toast')) {
        try { sessionStorage.setItem('xzj-pwd-toast', '1') } catch {}
        const { toast } = await import('./composables/useToast')
        toast.withAction('你还没有登录密码——设置后可用手机号或邮箱 + 密码登录', {
          label: '设置密码',
          onClick: () => profileRef.value?.openPwd()
        }, 8000)
      }
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

// 拼字邀请链接落地：/s/8621 重定向成 ?sprint=8621 —— 存本地后进编辑器自动弹「加入房间」
watch(() => route.query.sprint, (code) => {
  const c = String(code || '').trim()
  if (!c) return
  try { localStorage.setItem('xzj-sprint-invite', c) } catch {}
  toast.info(`🏁 收到拼字邀请：房间 ${c}——进一本书的编辑器，点底部 🏁 加入`)
  router.replace({ path: route.path, query: { ...route.query, sprint: undefined } }).catch(() => {})
}, { immediate: true })

const showQQModal = ref(false)
const onboardingRef = ref<{ start: () => void; forceStart: () => void } | null>(null)
const { canInstall, promptInstall } = usePwaInstall()
// 小程序 web-view 环境（UA 含 miniProgram）：收掉桌面向控件（安装到桌面/QQ群），顶栏只留 logo+书架+账号
const isMiniProgram = /miniProgram/i.test(navigator.userAgent)

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

// JWT payload 里的 exp（毫秒），解析失败返回 0
function jwtExp(t?: string | null): number {
  try {
    const payload = t!.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    return (JSON.parse(atob(payload)).exp || 0) * 1000
  } catch {
    return 0
  }
}
onMounted(() => {
  // 小程序 web-view 桥接：壳登录后把 token 拼在 URL（?xzj_at=…&xzj_rt=…），这里收进 localStorage 后清掉参数
  try {
    const q = new URLSearchParams(window.location.search)
    const at = q.get('xzj_at')
    const rt = q.get('xzj_rt')
    if (at) {
      // rt 一次性轮换：壳每次启动都拼登录时的老 token，无条件覆盖会把 web-view 里
      // 已轮换出的新 token 顶掉（旧 rt 已删库）→ 401「登录已过期」。只采用更新的。
      const stored = localStorage.getItem('access_token')
      if (!stored || jwtExp(at) > jwtExp(stored)) {
        localStorage.setItem('access_token', at)
        if (rt) localStorage.setItem('refresh_token', rt)
      }
      q.delete('xzj_at'); q.delete('xzj_rt')
      const clean = window.location.pathname + (q.toString() ? '?' + q.toString() : '') + window.location.hash
      window.history.replaceState(null, '', clean)
    }
  } catch { /* 忽略异常 URL */ }
  refreshLoginState()
  if (isLoggedIn.value) refreshProfile()
  document.addEventListener('contextmenu', (e) => e.preventDefault())
  window.addEventListener('replay-onboarding', onReplayOnboarding)
})
</script>

<style scoped>
.top-nav {
  flex-shrink: 0;
  height: calc(var(--nav-height) + var(--mp-safe-top, 0px));
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--mp-safe-top, 0px) calc(32px + var(--mp-capsule-pad, 0px)) 0 32px;
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
  /* 衬线字体(Noto Serif SC)部分环境下行高指标巨大，会把 normal 行高撑到 ~2.8em 顶出导航栏 */
  line-height: 1;
  height: 32px;
  /* 窄屏（小程序 web-view ~375px）下被 nav-right 挤压时，菌字会折到图标下方——禁止折行 */
  white-space: nowrap;
  flex-shrink: 0;
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

/* 窄屏适配：导航所有项单行排布，防止按钮内文字折行/整行换行 */
@media (max-width: 560px) {
  .top-nav { padding: var(--mp-safe-top, 0px) calc(14px + var(--mp-capsule-pad, 0px)) 0 14px; }
  .logo { font-size: 18px; gap: 6px; height: 28px; letter-spacing: 0; }
  .logo-icon { width: 22px; height: 22px; border-radius: 6px; }
  .nav-right { gap: 12px; }
  .nav-link { font-size: 13px; white-space: nowrap; padding: 4px 0; }
  .nav-nick { max-width: 4em; }
  .profile-link { min-width: 0; }
  /* 手机端放不下整排链接，QQ 按钮缩成「反馈」二字，否则右缘头像按钮被裁掉（同书架卡片 data-label 方案） */
  .qq-link { font-size: 0; }
  .qq-link::after { content: attr(data-label); font-size: 13px; }
}
@media (max-width: 380px) {
  .nav-nick { display: none; } /* 极窄屏只留头像+箭头 */
  .nav-right { gap: 10px; }
}
.nav-caret { font-size: 10px; color: var(--text-tertiary); }
.logout {
  font-family: inherit;
}

.install-link {
  color: var(--accent, #007aff);
  font-weight: 600;
}

/* 小程序左上角返回/主页：贴原生标题栏下沿的同色子栏——视觉上与标题连成一体（原生栏本身是平台保留区，塞不进去） */
.mp-nav-btns {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 36px;
  z-index: 120;
  display: flex;
  align-items: center;
  gap: 4px;
  padding-left: 14px;
  background: #f7f9f4; /* = 壳 navigationBarBackgroundColor，与原生栏无缝衔接 */
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}
.mp-nav-btn {
  width: 36px;
  height: 32px;
  border-radius: 8px;
  border: none;
  background: none;
  color: #333;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.mp-nav-btn:active { background: rgba(0, 0, 0, 0.06); }
/* 内容给顶部子栏让位（编辑器无子栏，不让位） */
.app-main.mp-pad { padding-top: 36px; box-sizing: border-box; }
.app-main.editor-mode.mp-pad { padding-top: 0; }

/* Toast 容器 */
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
  display: flex;
  align-items: center;
}

.toast-action {
  margin-left: 12px;
  padding: 2px 10px;
  border: 1px solid rgba(255, 255, 255, 0.65);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
}
.toast-action:hover { background: rgba(255, 255, 255, 0.3); }

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