<template>
  <div class="auth-page">
    <!-- 小程序 web-view 内：登录态过期，直接跳回壳的原生登录页（网页的微信入口在 web-view 里全是死路） -->
    <div v-if="isMiniProgram" class="auth-card" style="text-align:center">
      <p style="font-size:15px;color:var(--text-secondary);margin-bottom:20px">登录已过期</p>
      <button class="btn primary auth-submit" @click="backToMpLogin">重新登录</button>
    </div>
    <div v-else class="auth-card">
      <!-- 右上角扫码入口（微信内置浏览器里改为整宽一键登录按钮） -->
      <template v-if="!showReset">
        <div v-if="!isWechat" class="qr-corner" title="微信扫码登录" @click="showQr = true">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="#fff"><path d="M3 3h8v8H3V3zm2 2v4h4V5H5zm8-2h8v8h-8V3zm2 2v4h4V5h-4zM3 13h8v8H3v-8zm2 2v4h4v-4H5zm10-2h3v3h-3v-3zm-3 5h3v3h-3v-3zm5 0h3v3h-3v-3zm3-5h2v2h-2v-2zm0 7h2v2h-2v-2zm-9-2h2v2h-2v-2z"/></svg>
        </div>
        <div v-if="!isWechat" class="qr-hint">扫码登录</div>
        <button v-else type="button" class="btn wx-btn" @click="handleWxLogin">微信一键登录</button>
      </template>

      <div class="auth-header">
        <h1 class="logo">写作菌</h1>
        <p class="subtitle">登录账号继续创作</p>
      </div>

      <div v-if="!showReset" class="seg">
        <button :class="{ active: accountType === 'email' }" @click="switchAccountType('email')">邮箱</button>
        <button :class="{ active: accountType === 'phone' }" @click="switchAccountType('phone')">手机号</button>
      </div>

      <form v-if="!showReset" @submit.prevent="handleLogin" class="auth-form" novalidate>
        <div class="form-group">
          <input v-model.trim="account" type="text" :placeholder="accountType === 'email' ? '邮箱' : '手机号'" class="input" :class="{ invalid: touched.account && fieldErrors.account }" :autocomplete="accountType === 'email' ? 'email' : 'tel'" @blur="touched.account = true; validateLogin()" @input="validateLogin(false)" />
          <p v-if="touched.account && fieldErrors.account" class="field-error">{{ fieldErrors.account }}</p>
        </div>

        <div v-if="accountType === 'email' && mode === 'password'" class="form-group">
          <input v-model="password" type="password" placeholder="请输入密码" class="input" :class="{ invalid: touched.password && fieldErrors.password }" autocomplete="current-password" @blur="touched.password = true; validateLogin()" @input="validateLogin(false)" />
          <p v-if="touched.password && fieldErrors.password" class="field-error">{{ fieldErrors.password }}</p>
        </div>

        <div v-else class="form-group">
          <div class="code-row">
            <input v-model.trim="code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" class="input" :class="{ invalid: touched.code && fieldErrors.code }" @blur="touched.code = true; validateLogin()" @input="validateLogin(false)" />
            <button type="button" class="btn code-btn" :disabled="sendingCode || codeCountdown > 0 || !!fieldErrors.account" @click="sendCode">
              {{ codeCountdown > 0 ? `${codeCountdown}s` : (sendingCode ? '发送中' : '获取验证码') }}
            </button>
          </div>
          <p v-if="touched.code && fieldErrors.code" class="field-error">{{ fieldErrors.code }}</p>
        </div>

        <div v-if="accountType === 'email'" class="mode-switch">
          <a @click.prevent="toggleMode">{{ mode === 'password' ? '使用验证码登录 →' : '使用密码登录 →' }}</a>
        </div>

        <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
        <p v-if="successMsg" class="success-msg">{{ successMsg }}</p>
        <button type="submit" class="btn primary auth-submit" :disabled="loading || !canSubmit">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>

      <p v-if="!showReset" class="switch-text"><a href="#" @click.prevent="showReset = true; errorMsg = ''; successMsg = ''">忘记密码？</a></p>

      <!-- 密码重置 -->
      <form v-if="showReset" @submit.prevent="handleReset" class="auth-form" novalidate style="margin-top:16px">
        <div class="form-group">
          <label>邮箱</label>
          <input v-model.trim="resetEmail" type="email" placeholder="your@email.com" class="input" />
        </div>
        <div class="form-group">
          <label>验证码</label>
          <div class="code-row">
            <input v-model.trim="resetCode" inputmode="numeric" maxlength="6" placeholder="6 位验证码" class="input" />
            <button type="button" class="btn code-btn" :disabled="resetSending || resetCountdown > 0" @click="sendResetCode">
              {{ resetCountdown > 0 ? `${resetCountdown}s` : (resetSending ? '发送中' : '获取验证码') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>新密码</label>
          <input v-model="resetPassword" type="password" placeholder="至少 6 位" class="input" autocomplete="new-password" />
        </div>
        <button type="submit" class="btn primary auth-submit" :disabled="resetLoading">
          {{ resetLoading ? '重置中...' : '重置密码' }}
        </button>
        <p class="switch-text"><a href="#" @click.prevent="showReset = false; errorMsg = ''; successMsg = ''">返回登录</a></p>
      </form>

      <p v-if="!showReset" class="switch-text">还没有账号？ <router-link to="/register">立即注册</router-link></p>
      <p class="legal-note agree-row" @click="toggleAgree">
        <span class="agree-box" :class="{ on: agreed }"><Check v-if="agreed" :size="12"/></span>
        <span class="agree-text">我已阅读并同意</span>
        <router-link to="/agreement" target="_blank" @click.stop>《用户服务协议》</router-link>与
        <router-link to="/privacy" target="_blank" @click.stop>《隐私政策》</router-link>
      </p>

      <!-- 防刷点选验证（当日发送超阈值时触发）：弹窗，按提示条字的顺序点击大图 -->
      <div v-if="captcha" class="cap-overlay" @click.self="captcha = null">
        <div class="cap-dialog">
          <div class="cap-head"><span>安全验证</span><button class="cap-close" @click="captcha = null"><X :size="16"/></button></div>
          <div class="cap-body">
            <p class="sl-tip">按顺序点击大图中与<b>上方小条一致</b>的字</p>
            <div class="sl-stage">
              <img class="sl-thumb" :src="captcha.thumb" alt="" draggable="false" />
              <div class="sl-imgwrap">
                <img class="sl-bg" :src="captcha.img" alt="" draggable="false" @click="onImgClick" />
                <span v-for="(p, i) in clicks" :key="i" class="click-mark" :style="{ left: p[0] + 'px', top: p[1] + 'px' }">{{ i + 1 }}</span>
              </div>
            </div>
            <p class="sl-fail" v-if="clFail">点错了，按提示条的顺序再点</p>
            <a class="click-reset" v-if="clicks.length && clicks.length < captcha.n" @click.prevent="clicks = []">重新点</a>
          </div>
        </div>
      </div>

      <!-- 扫码浮层：iframe 嵌微信官方二维码页，登录成功后回调页 postMessage 回本页 -->
      <div v-if="showQr" class="qr-overlay">
        <button class="qr-close" @click="showQr = false">×</button>
        <h3>微信扫码登录</h3>
        <iframe class="qr-frame" :src="qrSrc" frameborder="0" scrolling="no"></iframe>
        <p class="qr-tip">打开<b>微信扫一扫</b>，确认即可登录<br>与小程序账号互通</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { userApi, projectApi } from '../core/api'
import { mpNavigate } from '../core/mp'
import { Check, X } from 'lucide-vue-next'

type LoginMode = 'password' | 'code'
type AccountType = 'email' | 'phone'
const router = useRouter()

// 登录后去向：?redirect= 回跳（登录墙保住的拼字邀请等深链）→ 有待消费的拼字邀请 → 直达最近一本书的编辑器（Editor 会自动弹「加入房间」）；否则进书架
async function afterLogin() {
  const redirect = String(router.currentRoute.value.query.redirect || '')
  if (redirect && redirect !== '/login' && redirect.startsWith('/')) {
    router.push(redirect)
    return
  }
  try {
    if (localStorage.getItem('xzj-sprint-invite')) {
      const res = await projectApi.list()
      const p = (res.data?.data || res.data || [])[0]
      if (p) {
        router.push(`/editor/${p.public_id || p.id}`)
        return
      }
    }
  } catch { /* 拉不到就照常进书架 */ }
  router.push('/bookshelf')
}
// 微信审核：协议默认不同意，用户自主勾选后才能登录/发码
const agreed = ref(false)
function toggleAgree() { agreed.value = !agreed.value }
function requireAgree(): boolean {
  if (!agreed.value) {
    errorMsg.value = '请先阅读并同意《用户服务协议》与《隐私政策》'
    return false
  }
  return true
}
// 小程序 web-view：登录态过期时回壳登录
const isMiniProgram = /miniProgram/i.test(navigator.userAgent) || (window as any).__wxjs_environment === 'miniProgram'
// 微信内置浏览器（非小程序）：整页授权一键登录
const isWechat = /MicroMessenger/i.test(navigator.userAgent) && !isMiniProgram
function backToMpLogin() {
  mpNavigate('/pages/login/login')
}
const handleWxLogin = () => {
  window.location.href = '/api/auth/wx-oauth/start?redirect=' + encodeURIComponent(window.location.origin + '/login')
}
// 扫码浮层
const showQr = ref(false)
const qrSrc = '/api/auth/wx-oauth/start?embed=1&redirect=' + encodeURIComponent(window.location.origin + '/login')
const onWxMsg = (e: MessageEvent) => {
  if (e.origin !== window.location.origin || !e.data || e.data.type !== 'xzj-wx-login') return
  if (e.data.at) {
    localStorage.setItem('access_token', e.data.at)
    if (e.data.rt) localStorage.setItem('refresh_token', e.data.rt)
    afterLogin()
  } else {
    errorMsg.value = e.data.error || '微信登录失败'
    showQr.value = false
  }
}
onMounted(() => window.addEventListener('message', onWxMsg))
onUnmounted(() => window.removeEventListener('message', onWxMsg))

const accountType = ref<AccountType>('email')
// 点选验证码（防刷）：按提示条字的顺序点击大图
type ClickCaptcha = { id: string; img: string; thumb: string; n: number; w: number; h: number }
const captcha = ref<ClickCaptcha | null>(null)
const clicks = ref<[number, number][]>([])
const clFail = ref(false)
let clickT0 = 0

function onImgClick(e: MouseEvent) {
  if (!captcha.value || clicks.value.length >= captcha.value.n) return
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const scaleX = captcha.value.w / rect.width
  const scaleY = captcha.value.h / rect.height
  if (!clicks.value.length) { clickT0 = performance.now(); clFail.value = false }
  clicks.value.push([Math.round((e.clientX - rect.left) * scaleX), Math.round((e.clientY - rect.top) * scaleY)])
  if (clicks.value.length === captcha.value.n) {
    setTimeout(() => sendCode(), 150) // 最后一个点的标记渲染完再提交
  }
}
const mode = ref<LoginMode>('password')
const account = ref('')
const password = ref('')
const code = ref('')
const loading = ref(false)
const sendingCode = ref(false)
const codeCountdown = ref(0)
const errorMsg = ref('')
const successMsg = ref('')
const fieldErrors = reactive({ account: '', password: '', code: '' })
const touched = reactive({ account: false, password: false, code: false })
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const phonePattern = /^1[3-9]\d{9}$/

function switchAccountType(t: AccountType) {
  accountType.value = t
  errorMsg.value = ''
  successMsg.value = ''
  touched.account = false
  fieldErrors.account = ''
  // 手机号只有验证码登录
  if (t === 'phone') mode.value = 'code'
  validateLogin(false)
}

function toggleMode() {
  mode.value = mode.value === 'password' ? 'code' : 'password'
  errorMsg.value = ''
}

// 密码重置
const showReset = ref(false)
const resetEmail = ref('')
const resetCode = ref('')
const resetPassword = ref('')
const resetSending = ref(false)
const resetCountdown = ref(0)
const resetLoading = ref(false)

const canSubmit = computed(() => {
  if (fieldErrors.account || !account.value.trim()) return false
  if (accountType.value === 'phone') return /^\d{6}$/.test(code.value) && !fieldErrors.code
  if (mode.value === 'password') return !!password.value && !fieldErrors.password
  return /^\d{6}$/.test(code.value) && !fieldErrors.code
})

function normalizeError(err: any) {
  const raw = String(err.response?.data?.error || '')
  if (raw.includes('email not registered')) return '这个邮箱还没有注册'
  if (raw.includes('invalid or expired code')) return '验证码错误或已过期'
  if (raw.includes('send too frequently')) return '验证码发送太频繁，请稍后再试'
  if (err.response?.status === 429 || raw.includes('send limit exceeded')) return '发送次数已达上限，请稍后再试'
  if (raw.includes('sms not configured')) return '手机号登录暂未开放，请用邮箱或扫码登录'
  if (raw.includes('invalid phone')) return '请输入正确的手机号'
  if (err.response?.status === 401 || raw.includes('invalid email or password')) return '邮箱或密码不正确'
  if (raw.includes('required') || raw.includes('email')) return '请输入正确的账号'
  return raw || '操作失败，请稍后再试'
}

function validateLogin(showAll = false) {
  if (showAll) {
    touched.account = true
    touched.password = accountType.value === 'email' && mode.value === 'password'
    touched.code = accountType.value === 'phone' || mode.value === 'code'
  }
  fieldErrors.account = ''
  fieldErrors.password = ''
  fieldErrors.code = ''
  const acct = account.value.trim()
  if (!acct) fieldErrors.account = accountType.value === 'email' ? '请输入邮箱' : '请输入手机号'
  else if (accountType.value === 'email' && !emailPattern.test(acct)) fieldErrors.account = '请输入正确的邮箱格式'
  else if (accountType.value === 'phone' && !phonePattern.test(acct)) fieldErrors.account = '请输入正确的手机号'
  if (accountType.value === 'email' && mode.value === 'password' && !password.value) fieldErrors.password = '请输入密码'
  if ((accountType.value === 'phone' || mode.value === 'code') && code.value && !/^\d{6}$/.test(code.value)) fieldErrors.code = '请输入 6 位验证码'
  return !fieldErrors.account && !fieldErrors.password && !fieldErrors.code
}

function startCountdown() {
  codeCountdown.value = 60
  const timer = window.setInterval(() => {
    codeCountdown.value -= 1
    if (codeCountdown.value <= 0) window.clearInterval(timer)
  }, 1000)
}

async function sendCode() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!requireAgree()) return
  touched.account = true
  validateLogin(false)
  if (fieldErrors.account) return
  sendingCode.value = true
  try {
    const extra = captcha.value && captcha.value.n && clicks.value.length === captcha.value.n
      ? { captcha_id: captcha.value.id, captcha_answer: JSON.stringify({ points: clicks.value, dur: Math.round(performance.now() - clickT0) }) }
      : {}
    if (accountType.value === 'phone') {
      const res = await userApi.sendSmsCode({ phone: account.value.trim(), scene: 'login', ...extra })
      const c = res.data?.data?.code
      successMsg.value = c ? `验证码已发送（测试码：${c}）` : '验证码已发送，请查看短信'
    } else {
      await userApi.sendEmailCode({ email: account.value.trim(), scene: 'login' })
      successMsg.value = '验证码已发送，请查看邮箱'
    }
    captcha.value = null
    clicks.value = []
    clFail.value = false
    startCountdown()
  } catch (err: any) {
    const r = err.response
    if ((r?.status === 412 || r?.status === 428) && r.data?.captcha) {
      clFail.value = clicks.value.length === 3 // 刚点满 3 个还失败 = 顺序/位置错
      captcha.value = r.data.captcha
      clicks.value = []
    } else {
      captcha.value = null
      errorMsg.value = normalizeError(err)
    }
  } finally {
    sendingCode.value = false
  }
}

async function handleLogin() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!requireAgree()) return
  if (!validateLogin(true)) return
  loading.value = true
  try {
    const res = accountType.value === 'phone'
      ? await userApi.smsLogin({ phone: account.value.trim(), code: code.value })
      : await userApi.login(mode.value === 'password'
        ? { email: account.value.trim(), password: password.value }
        : { email: account.value.trim(), code: code.value })
    const data = res.data.data || res.data
    if (data.access_token) localStorage.setItem('access_token', data.access_token)
    if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
    afterLogin()
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    loading.value = false
  }
}

// 密码重置
async function sendResetCode() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!resetEmail.value.trim() || !emailPattern.test(resetEmail.value.trim())) {
    errorMsg.value = '请输入正确的邮箱'
    return
  }
  resetSending.value = true
  try {
    await userApi.sendEmailCode({ email: resetEmail.value.trim(), scene: 'reset_password' })
    successMsg.value = '验证码已发送，请查看邮箱'
    resetCountdown.value = 60
    const timer = window.setInterval(() => {
      resetCountdown.value -= 1
      if (resetCountdown.value <= 0) window.clearInterval(timer)
    }, 1000)
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    resetSending.value = false
  }
}

async function handleReset() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!resetEmail.value.trim() || !/^\d{6}$/.test(resetCode.value) || resetPassword.value.length < 6) {
    errorMsg.value = '请填写邮箱、6位验证码和至少6位新密码'
    return
  }
  resetLoading.value = true
  try {
    await userApi.resetPassword({
      email: resetEmail.value.trim(),
      code: resetCode.value,
      password: resetPassword.value
    })
    successMsg.value = '密码已重置，请用新密码登录'
    showReset.value = false
    resetEmail.value = ''
    resetCode.value = ''
    resetPassword.value = ''
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    resetLoading.value = false
  }
}
</script>

<style scoped>
.auth-page { display:flex; align-items:center; justify-content:center; min-height:100vh; background:var(--bg-primary); }
.auth-card { background:var(--bg-secondary); padding:48px 40px; border-radius:var(--radius-lg); box-shadow:var(--shadow-xl); width:420px; max-width:90vw; position:relative; }
.auth-header { text-align:center; margin-bottom:24px; }
.logo { font-family:var(--font-serif); font-size:36px; font-weight:700; color:var(--text-primary); margin-bottom:8px; letter-spacing:2px; }
.subtitle { color:var(--text-secondary); font-size:14px; }
.seg { display:flex; gap:4px; background:var(--bg-tertiary,#f5f5f7); padding:4px; border-radius:10px; margin-bottom:22px; }
.seg button { flex:1; border:0; background:transparent; padding:8px; border-radius:8px; cursor:pointer; font-size:13px; color:var(--text-secondary); }
.seg button.active { background:var(--bg-secondary); color:var(--text-primary); font-weight:600; box-shadow:0 1px 4px var(--shadow-color); }
.form-group { margin-bottom:16px; }
.form-group label { display:block; font-size:13px; font-weight:500; color:var(--text-secondary); margin-bottom:6px; }
.input.invalid { border-color:#ff3b30; background:rgba(255,59,48,.04); }
.code-row { display:flex; gap:8px; }
.code-row .input { flex:1; }
.code-btn { white-space:nowrap; padding:0 12px; }
.mode-switch { text-align:right; margin:-6px 0 14px; }
.mode-switch a { font-size:12px; color:var(--accent,#007aff); cursor:pointer; }
.field-error { color:#ff3b30; font-size:12px; line-height:1.4; margin-top:6px; }
.error-msg,.success-msg { border-radius:var(--radius-sm); padding:8px 10px; font-size:13px; margin-bottom:12px; }
.error-msg { color:#ff3b30; background:rgba(255,59,48,.08); border:1px solid rgba(255,59,48,.16); }
.success-msg { color:#248a3d; background:rgba(52,199,89,.10); border:1px solid rgba(52,199,89,.18); }
.auth-submit { width:100%; padding:12px; font-size:15px; margin-top:8px; }
.legal-note { font-size:12px; color:var(--text-secondary); text-align:center; margin-top:14px; }
.legal-note a { color:var(--text-secondary); text-decoration:underline; }
.agree-row { display:flex; align-items:center; justify-content:center; gap:5px; flex-wrap:wrap; cursor:pointer; }
.agree-box { width:16px; height:16px; border:1.5px solid #b8c2ba; border-radius:4px; font-size:11px; line-height:14px; text-align:center; color:#fff; flex-shrink:0; }
.agree-box.on { background:#6aa877; border-color:#6aa877; }
.agree-text { color:var(--text-secondary); }
.auth-submit:disabled,.code-btn:disabled { opacity:.5; cursor:not-allowed; }
.wx-btn {
  width:100%; padding:12px; font-size:15px; font-weight:600; margin-bottom:16px;
  background:#07c160; color:#fff; border:none; border-radius:10px; cursor:pointer;
}
.wx-btn:active { opacity:.85; }
.qr-corner {
  position:absolute; top:0; right:0; width:64px; height:64px; cursor:pointer;
  border-radius:0 var(--radius-lg) 0 var(--radius-lg);
  background:linear-gradient(225deg, #07c160 50%, transparent 50%);
  transition:transform .15s; display:flex; align-items:flex-start; justify-content:flex-end;
  padding:9px 8px 0 0;
}
.qr-corner:hover { transform:scale(1.06); }
.qr-hint { position:absolute; top:66px; right:12px; font-size:11px; color:var(--text-tertiary,#999); }
.qr-overlay {
  position:absolute; inset:0; background:var(--bg-secondary); border-radius:var(--radius-lg); z-index:5;
  display:flex; flex-direction:column; align-items:center; justify-content:center; padding:20px;
}
.qr-overlay h3 { font-size:17px; color:var(--text-primary); margin-bottom:16px; }
.qr-close {
  position:absolute; top:14px; right:18px; font-size:22px; color:var(--text-tertiary,#999);
  background:none; border:0; cursor:pointer;
}
.qr-frame { width:300px; height:340px; border:none; }
.qr-tip { font-size:13px; color:var(--text-secondary); margin-top:14px; text-align:center; line-height:1.8; }
.qr-tip b { color:#07c160; }
.cap-overlay {
  position:fixed; inset:0; background:rgba(0,0,0,.45); z-index:300;
  display:flex; align-items:center; justify-content:center; padding:16px;
}
.cap-dialog {
  background:var(--bg-secondary,#fff); border-radius:20px; padding:0 0 18px;
  box-shadow:0 24px 64px rgba(0,0,0,.2); max-width:92vw;
}
.cap-head {
  display:flex; justify-content:space-between; align-items:center;
  padding:14px 16px 4px; font-weight:600; font-size:15px; color:var(--text-primary);
}
.cap-close { border:none; background:none; cursor:pointer; font-size:14px; color:var(--text-tertiary); width:28px; height:28px; border-radius:50%; }
.cap-close:hover { background:var(--bg-tertiary,#f5f5f7); }
.cap-body { padding:8px 20px 0; display:flex; flex-direction:column; align-items:center; }
.sl-tip { font-size:13px; color:var(--text-secondary); margin-bottom:12px; }
.click-hint { color:var(--accent,#007aff); font-size:15px; letter-spacing:2px; }
.sl-stage { width:280px; user-select:none; }
.sl-thumb { display:block; width:170px; height:46px; margin:0 auto 10px; border-radius:8px; border:1px solid var(--border,#e5e5eb); }
.sl-imgwrap { position:relative; }
.sl-bg { display:block; width:280px; height:200px; border-radius:10px; cursor:crosshair; }
.click-mark {
  position:absolute; width:18px; height:18px; margin:-9px 0 0 -9px; border-radius:50%;
  background:#ff3b30; color:#fff; font-size:11px; line-height:18px; text-align:center;
  box-shadow:0 0 0 2px #fff; pointer-events:none;
}
.sl-fail { color:#ff3b30; font-size:12px; margin-top:10px; }
.click-reset { font-size:12px; color:var(--accent,#007aff); margin-top:8px; cursor:pointer; }
.switch-text { text-align:center; margin-top:24px; color:var(--text-secondary); font-size:14px; }
</style>
