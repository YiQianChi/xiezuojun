<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <h1 class="logo">写作菌</h1>
        <p class="subtitle">注册账号开始创作</p>
      </div>

      <form @submit.prevent="handleRegister" class="auth-form" novalidate>
        <div class="form-group">
          <label>邮箱</label>
          <input v-model.trim="email" type="email" placeholder="your@email.com" class="input" :class="{ invalid: touched.email && fieldErrors.email }" autocomplete="email" @blur="touched.email = true; validateRegister()" @input="validateRegister(false)" />
          <p v-if="touched.email && fieldErrors.email" class="field-error">{{ fieldErrors.email }}</p>
        </div>

        <div class="form-group">
          <label>邮箱验证码</label>
          <div class="code-row">
            <input v-model.trim="code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" class="input" :class="{ invalid: touched.code && fieldErrors.code }" @blur="touched.code = true; validateRegister()" @input="validateRegister(false)" />
            <button type="button" class="btn code-btn" :disabled="sendingCode || codeCountdown > 0 || !!fieldErrors.email" @click="sendCode">
              {{ codeCountdown > 0 ? `${codeCountdown}s` : (sendingCode ? '发送中' : '获取验证码') }}
            </button>
          </div>
          <p v-if="touched.code && fieldErrors.code" class="field-error">{{ fieldErrors.code }}</p>
        </div>

        <div class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="至少 6 位" class="input" :class="{ invalid: touched.password && fieldErrors.password }" autocomplete="new-password" @blur="touched.password = true; validateRegister()" @input="validateRegister(false)" />
          <p v-if="touched.password && fieldErrors.password" class="field-error">{{ fieldErrors.password }}</p>
        </div>

        <div class="form-group">
          <label>确认密码</label>
          <input v-model="confirmPassword" type="password" placeholder="再次输入密码" class="input" :class="{ invalid: touched.confirmPassword && fieldErrors.confirmPassword }" autocomplete="new-password" @blur="touched.confirmPassword = true; validateRegister()" @input="validateRegister(false)" />
          <p v-if="touched.confirmPassword && fieldErrors.confirmPassword" class="field-error">{{ fieldErrors.confirmPassword }}</p>
        </div>

        <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
        <p v-if="successMsg" class="success-msg">{{ successMsg }}</p>
        <button type="submit" class="btn primary auth-submit" :disabled="loading || !canSubmit">
          {{ loading ? '注册中...' : '注册' }}
        </button>
      </form>

      <p class="switch-text">已有账号？ <router-link to="/login">立即登录</router-link></p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'

const router = useRouter()
const email = ref('')
const code = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const sendingCode = ref(false)
const codeCountdown = ref(0)
const errorMsg = ref('')
const successMsg = ref('')
const fieldErrors = reactive({ email: '', code: '', password: '', confirmPassword: '' })
const touched = reactive({ email: false, code: false, password: false, confirmPassword: false })
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const canSubmit = computed(() => {
  return !!email.value.trim() && /^\d{6}$/.test(code.value) && !!password.value && !!confirmPassword.value
    && !fieldErrors.email && !fieldErrors.code && !fieldErrors.password && !fieldErrors.confirmPassword
})

function normalizeError(err: any) {
  const raw = String(err.response?.data?.error || '')
  if (raw.includes('email already registered')) return '这个邮箱已经注册过了，请直接登录'
  if (raw.includes('invalid or expired code')) return '验证码错误或已过期'
  if (raw.includes('send too frequently')) return '验证码发送太频繁，请稍后再试'
  if (raw.includes('min=6')) return '密码至少需要 6 位'
  if (raw.includes('required') || raw.includes('email')) return '请输入正确的邮箱和验证码'
  return raw || '注册失败，请稍后再试'
}

function validateRegister(showAll = false) {
  if (showAll) {
    touched.email = true
    touched.code = true
    touched.password = true
    touched.confirmPassword = true
  }
  fieldErrors.email = ''
  fieldErrors.code = ''
  fieldErrors.password = ''
  fieldErrors.confirmPassword = ''

  const mail = email.value.trim()
  if (!mail) fieldErrors.email = '请输入邮箱'
  else if (!emailPattern.test(mail)) fieldErrors.email = '请输入正确的邮箱格式'
  if (!/^\d{6}$/.test(code.value)) fieldErrors.code = '请输入 6 位验证码'
  if (!password.value) fieldErrors.password = '请输入密码'
  else if (password.value.length < 6) fieldErrors.password = '密码至少需要 6 位'
  if (!confirmPassword.value) fieldErrors.confirmPassword = '请再次输入密码'
  else if (password.value && confirmPassword.value !== password.value) fieldErrors.confirmPassword = '两次密码不一致'
  return !fieldErrors.email && !fieldErrors.code && !fieldErrors.password && !fieldErrors.confirmPassword
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
  validateRegister(false)
  if (fieldErrors.email) return
  sendingCode.value = true
  try {
    await userApi.sendEmailCode({ email: email.value.trim(), scene: 'register' })
    successMsg.value = '验证码已发送，请查看邮箱'
    startCountdown()
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    sendingCode.value = false
  }
}

async function handleRegister() {
  errorMsg.value = ''
  successMsg.value = ''
  if (!validateRegister(true)) return
  loading.value = true
  try {
    const res = await userApi.register({ email: email.value.trim(), password: password.value, code: code.value })
    const data = res.data.data || res.data
    if (data.access_token) localStorage.setItem('access_token', data.access_token)
    if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
    localStorage.setItem('xiezuojun_onboarded', '0') // 标记需要引导
    successMsg.value = '注册成功，正在跳转...'
    setTimeout(() => router.push('/bookshelf'), 600)
  } catch (err: any) {
    errorMsg.value = normalizeError(err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page { display:flex; align-items:center; justify-content:center; min-height:100vh; background:var(--bg-primary); }
.auth-card { background:var(--bg-secondary); padding:48px 40px; border-radius:var(--radius-lg); box-shadow:var(--shadow-xl); width:420px; max-width:90vw; }
.auth-header { text-align:center; margin-bottom:32px; }
.logo { font-family:var(--font-serif); font-size:36px; font-weight:700; color:var(--text-primary); margin-bottom:8px; letter-spacing:2px; }
.subtitle { color:var(--text-secondary); font-size:14px; }
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
