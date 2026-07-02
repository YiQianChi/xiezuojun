<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <h1 class="logo">写作菌</h1>
        <p class="subtitle">登录账号继续创作</p>
      </div>

      <div v-if="!showReset" class="login-tabs">
        <button :class="{ active: mode === 'password' }" @click="mode = 'password'; errorMsg = ''">密码登录</button>
        <button :class="{ active: mode === 'code' }" @click="mode = 'code'; errorMsg = ''">验证码登录</button>
      </div>

      <form v-if="!showReset" @submit.prevent="handleLogin" class="auth-form" novalidate>
        <div class="form-group">
          <label>邮箱</label>
          <input v-model.trim="email" type="email" placeholder="your@email.com" class="input" :class="{ invalid: touched.email && fieldErrors.email }" autocomplete="email" @blur="touched.email = true; validateLogin()" @input="validateLogin(false)" />
          <p v-if="touched.email && fieldErrors.email" class="field-error">{{ fieldErrors.email }}</p>
        </div>

        <div v-if="mode === 'password'" class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="请输入密码" class="input" :class="{ invalid: touched.password && fieldErrors.password }" autocomplete="current-password" @blur="touched.password = true; validateLogin()" @input="validateLogin(false)" />
          <p v-if="touched.password && fieldErrors.password" class="field-error">{{ fieldErrors.password }}</p>
        </div>

        <div v-else class="form-group">
          <label>邮箱验证码</label>
          <div class="code-row">
            <input v-model.trim="code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" class="input" :class="{ invalid: touched.code && fieldErrors.code }" @blur="touched.code = true; validateLogin()" @input="validateLogin(false)" />
            <button type="button" class="btn code-btn" :disabled="sendingCode || codeCountdown > 0 || !!fieldErrors.email" @click="sendCode">
              {{ codeCountdown > 0 ? `${codeCountdown}s` : (sendingCode ? '发送中' : '获取验证码') }}
            </button>
          </div>
          <p v-if="touched.code && fieldErrors.code" class="field-error">{{ fieldErrors.code }}</p>
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'

type LoginMode = 'password' | 'code'
const router = useRouter()
const mode = ref<LoginMode>('password')
const email = ref('')
const password = ref('')
const code = ref('')
const loading = ref(false)
const sendingCode = ref(false)
const codeCountdown = ref(0)
const errorMsg = ref('')
const successMsg = ref('')
const fieldErrors = reactive({ email: '', password: '', code: '' })
const touched = reactive({ email: false, password: false, code: false })
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// 密码重置
const showReset = ref(false)
const resetEmail = ref('')
const resetCode = ref('')
const resetPassword = ref('')
const resetSending = ref(false)
const resetCountdown = ref(0)
const resetLoading = ref(false)

const canSubmit = computed(() => {
  if (!email.value.trim() || fieldErrors.email) return false
  if (mode.value === 'password') return !!password.value && !fieldErrors.password
  return /^\d{6}$/.test(code.value) && !fieldErrors.code
})

function normalizeError(err: any) {
  const raw = String(err.response?.data?.error || '')
  if (raw.includes('email not registered')) return '这个邮箱还没有注册'
  if (raw.includes('invalid or expired code')) return '验证码错误或已过期'
  if (raw.includes('send too frequently')) return '验证码发送太频繁，请稍后再试'
  if (err.response?.status === 401 || raw.includes('invalid email or password')) return '邮箱或密码不正确'
  if (raw.includes('required') || raw.includes('email')) return '请输入正确的邮箱'
  return raw || '操作失败，请稍后再试'
}

function validateLogin(showAll = false) {
  if (showAll) {
    touched.email = true
    touched.password = mode.value === 'password'
    touched.code = mode.value === 'code'
  }
  fieldErrors.email = ''
  fieldErrors.password = ''
  fieldErrors.code = ''
  const mail = email.value.trim()
  if (!mail) fieldErrors.email = '请输入邮箱'
  else if (!emailPattern.test(mail)) fieldErrors.email = '请输入正确的邮箱格式'
  if (mode.value === 'password' && !password.value) fieldErrors.password = '请输入密码'
  if (mode.value === 'code' && !/^\d{6}$/.test(code.value)) fieldErrors.code = '请输入 6 位验证码'
  return !fieldErrors.email && !fieldErrors.password && !fieldErrors.code
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
  touched.email = true
  validateLogin(false)
  if (fieldErrors.email) return
  sendingCode.value = true
  try {
    await userApi.sendEmailCode({ email: email.value.trim(), scene: 'login' })
    successMsg.value = '验证码已发送，请查看邮箱'
    startCountdown()
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    sendingCode.value = false
  }
}

async function handleLogin() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!validateLogin(true)) return
  loading.value = true
  try {
    const payload = mode.value === 'password'
      ? { email: email.value.trim(), password: password.value }
      : { email: email.value.trim(), code: code.value }
    const res = await userApi.login(payload)
    const data = res.data.data || res.data
    if (data.access_token) localStorage.setItem('access_token', data.access_token)
    if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
    router.push('/bookshelf')
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
.auth-page { display:flex; align-items:center; justify-content:center; min-height:100vh; background:linear-gradient(135deg,#f5f5f7 0%,#e8e8ed 100%); }
.auth-card { background:var(--bg-secondary); padding:48px 40px; border-radius:var(--radius-lg); box-shadow:var(--shadow-xl); width:420px; max-width:90vw; }
.auth-header { text-align:center; margin-bottom:24px; }
.logo { font-family:var(--font-serif); font-size:36px; font-weight:700; color:var(--text-primary); margin-bottom:8px; letter-spacing:2px; }
.subtitle { color:var(--text-secondary); font-size:14px; }
.login-tabs { display:flex; gap:8px; background:var(--bg-tertiary,#f5f5f7); padding:4px; border-radius:10px; margin-bottom:22px; }
.login-tabs button { flex:1; border:0; background:transparent; padding:8px; border-radius:8px; cursor:pointer; color:var(--text-secondary); }
.login-tabs button.active { background:#fff; color:var(--accent,#007aff); box-shadow:0 1px 4px rgba(0,0,0,.08); }
.form-group { margin-bottom:16px; }
.form-group label { display:block; font-size:13px; font-weight:500; color:var(--text-secondary); margin-bottom:6px; }
.input.invalid { border-color:#ff3b30; background:rgba(255,59,48,.04); }
.code-row { display:flex; gap:8px; }
.code-row .input { flex:1; }
.code-btn { white-space:nowrap; padding:0 12px; }
.field-error { color:#ff3b30; font-size:12px; line-height:1.4; margin-top:6px; }
.error-msg,.success-msg { border-radius:var(--radius-sm); padding:8px 10px; font-size:13px; margin-bottom:12px; }
.error-msg { color:#ff3b30; background:rgba(255,59,48,.08); border:1px solid rgba(255,59,48,.16); }
.success-msg { color:#248a3d; background:rgba(52,199,89,.10); border:1px solid rgba(52,199,89,.18); }
.auth-submit { width:100%; padding:12px; font-size:15px; margin-top:8px; }
.auth-submit:disabled,.code-btn:disabled { opacity:.5; cursor:not-allowed; }
.switch-text { text-align:center; margin-top:24px; color:var(--text-secondary); font-size:14px; }
</style>
