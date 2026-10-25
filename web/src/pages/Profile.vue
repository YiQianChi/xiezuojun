<template>
  <div class="profile-page">
    <div class="profile-card">
      <h2>个人设置</h2>

      <!-- 资料区 -->
      <section class="sec">
        <h3>基本资料</h3>
        <div class="avatar-row">
          <div class="avatar" :style="avatarStyle">{{ initials }}</div>
          <div class="avatar-ops">
            <div class="emoji-picker">
              <button v-for="e in EMOJIS" :key="e" class="emoji-btn" :class="{ on: form.avatar === e }" @click="form.avatar = e">{{ e }}</button>
            </div>
            <button v-if="form.avatar" class="btn sm" @click="form.avatar = ''">清除头像</button>
          </div>
        </div>
        <label class="field">
          <span>昵称</span>
          <input v-model="form.nickname" class="input" placeholder="给自己起个笔名" maxlength="20" />
        </label>
        <label class="field">
          <span>邮箱</span>
          <input :value="me.email" class="input" disabled />
        </label>
        <label class="field">
          <span>套餐</span>
          <input :value="PLAN_NAMES[me.plan] || me.plan" class="input" disabled />
        </label>
        <button class="btn primary" @click="saveProfile" :disabled="saving">{{ saving ? '保存中…' : '保存资料' }}</button>
      </section>

      <!-- 改密码 -->
      <section class="sec">
        <h3>修改密码</h3>
        <label class="field">
          <span>旧密码</span>
          <input v-model="pwd.old" type="password" class="input" placeholder="当前密码" />
        </label>
        <label class="field">
          <span>新密码</span>
          <input v-model="pwd.next" type="password" class="input" placeholder="至少 6 位" />
        </label>
        <label class="field">
          <span>确认新密码</span>
          <input v-model="pwd.confirm" type="password" class="input" placeholder="再输一遍" />
        </label>
        <button class="btn primary" @click="changePwd" :disabled="changingPwd">{{ changingPwd ? '提交中…' : '修改密码' }}</button>
      </section>

      <!-- 我的统计 -->
      <section class="sec" v-if="usage">
        <h3>我的写作</h3>
        <div class="usage-grid">
          <div><b>{{ usage.total_words }}</b><span>累计字数</span></div>
          <div><b>{{ usage.month_words }}</b><span>本月字数</span></div>
          <div><b>{{ usage.today_words }}</b><span>今日字数</span></div>
        </div>
      </section>

      <section class="sec danger-sec">
        <h3>账号</h3>
        <button class="btn logout-btn" @click="doLogout">退出登录</button>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const EMOJIS = ['✍️', '📖', '🐉', '🗡️', '🌸', '🌙', '⭐', '🔥', '🦊', '🐱', '🐼', '🧙']
const PLAN_NAMES: Record<string, string> = { free: '免费版', vip: 'VIP', svip: 'SVIP' }

const me = ref<any>({})
const form = ref({ nickname: '', avatar: '' })
const pwd = ref({ old: '', next: '', confirm: '' })
const usage = ref<any>(null)
const saving = ref(false)
const changingPwd = ref(false)

const initials = computed(() => {
  const a = form.value.avatar
  if (a) return ''
  const n = form.value.nickname || me.value.email || '?'
  return n.slice(0, 1).toUpperCase()
})
const avatarStyle = computed(() => form.value.avatar
  ? { background: 'var(--bg-tertiary, #f0f0f3)', fontSize: '30px', lineHeight: '64px' }
  : { background: 'var(--accent, #007aff)', color: '#fff', fontSize: '26px', fontWeight: 700, lineHeight: '64px' })

onMounted(async () => {
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
    form.value.nickname = me.value.nickname || ''
    form.value.avatar = me.value.avatar || ''
  } catch {}
  try {
    const u = await userApi.usage()
    usage.value = u.data?.data
  } catch {}
})

async function saveProfile() {
  saving.value = true
  try {
    const payload: any = {}
    if (form.value.nickname !== (me.value.nickname || '')) payload.nickname = form.value.nickname
    if (form.value.avatar !== (me.value.avatar || '')) payload.avatar = form.value.avatar
    if (!Object.keys(payload).length) { toast.info('没有修改'); return }
    await userApi.updateProfile(payload)
    toast.success('资料已保存')
    me.value.nickname = form.value.nickname
    me.value.avatar = form.value.avatar
    // 导航栏昵称头像跟随
    try { localStorage.setItem('xzj-nickname', form.value.nickname); localStorage.setItem('xzj-avatar', form.value.avatar) } catch {}
    window.dispatchEvent(new CustomEvent('xzj-profile-updated'))
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function changePwd() {
  const { old: o, next: n, confirm: c } = pwd.value
  if (!o || !n) { toast.error('请填写完整'); return }
  if (n.length < 6) { toast.error('新密码至少 6 位'); return }
  if (n !== c) { toast.error('两次新密码不一致'); return }
  changingPwd.value = true
  try {
    await userApi.changePassword({ old_password: o, new_password: n })
    toast.success('密码已修改')
    pwd.value = { old: '', next: '', confirm: '' }
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

function doLogout() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  router.push('/login')
}
</script>

<style scoped>
.profile-page { min-height: 100vh; background: var(--bg-primary, #f6f6f8); padding: 28px 16px 60px; }
.profile-card { max-width: 560px; margin: 0 auto; }
h2 { font-size: 20px; color: var(--text-primary); margin: 0 0 18px; }
.sec {
  background: var(--bg-secondary, #fff); border-radius: 14px; padding: 18px 20px;
  border: 1px solid var(--border, #eee); margin-bottom: 16px;
}
.sec h3 { font-size: 15px; color: var(--text-primary); margin: 0 0 14px; }
.avatar-row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 14px; }
.avatar {
  width: 64px; height: 64px; border-radius: 50%; text-align: center; flex-shrink: 0;
  overflow: hidden;
}
.avatar-ops { flex: 1; }
.emoji-picker { display: flex; flex-wrap: wrap; gap: 6px; }
.emoji-btn {
  width: 36px; height: 36px; border-radius: 10px; border: 1px solid var(--border, #e5e5ea);
  background: var(--bg-secondary, #fff); font-size: 18px; cursor: pointer;
}
.emoji-btn.on { border-color: var(--accent, #007aff); background: rgba(0,122,255,.08); }
.avatar-ops .btn { margin-top: 8px; }
.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px; }
.field span { font-size: 13px; color: var(--text-secondary); }
.input {
  padding: 9px 12px; border: 1px solid var(--border, #e5e5ea); border-radius: 8px;
  font-size: 14px; outline: none; background: var(--bg-secondary, #fff); color: var(--text-primary);
}
.input:disabled { color: var(--text-tertiary); background: var(--bg-tertiary, #f5f5f7); }
.input:focus { border-color: var(--accent, #007aff); }
.btn {
  border: 1px solid var(--border, #e0e0e0); background: transparent; padding: 9px 16px;
  border-radius: 8px; cursor: pointer; font-size: 14px; color: var(--text-primary);
}
.btn.primary { background: var(--accent, #007aff); color: #fff; border-color: var(--accent, #007aff); }
.btn.primary:disabled { opacity: .6; cursor: not-allowed; }
.btn.sm { padding: 5px 10px; font-size: 12px; }
.usage-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.usage-grid > div { background: var(--bg-tertiary, #f5f5f7); border-radius: 10px; padding: 12px 6px; text-align: center; }
.usage-grid b { display: block; font-size: 19px; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.usage-grid span { font-size: 11px; color: var(--text-tertiary); }
.logout-btn { color: #d70015; border-color: rgba(255,59,48,.3); }
.logout-btn:hover { background: rgba(255,59,48,.06); }
</style>
