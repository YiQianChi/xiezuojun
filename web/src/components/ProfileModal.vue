<template>
  <Teleport to="body">
    <div v-if="open" class="pm-overlay" @click.self="$emit('update:open', false)">
      <div class="pm-modal">
        <div class="pm-head">
          <span>个人设置</span>
          <button class="pm-close" @click="$emit('update:open', false)">✕</button>
        </div>
        <div class="pm-body">
          <!-- 资料区 -->
          <div class="avatar-row">
            <div class="avatar" :style="avatarStyle">{{ initials }}</div>
            <div class="avatar-ops">
              <div class="emoji-picker">
                <button v-for="e in EMOJIS" :key="e" class="emoji-btn" :class="{ on: form.avatar === e }" @click="form.avatar = e">{{ e }}</button>
                <button v-if="form.avatar" class="emoji-btn emoji-clear" @click="form.avatar = ''">✕</button>
              </div>
            </div>
          </div>
          <label class="field">
            <span>昵称</span>
            <input v-model="form.nickname" class="input" placeholder="给自己起个笔名" maxlength="20" @keyup.enter="saveProfile" />
          </label>
          <div class="meta-row"><span>邮箱</span><b>{{ me.email }}</b></div>
          <div class="meta-row"><span>套餐</span><b>{{ PLAN_NAMES[me.plan] || me.plan }}</b></div>
          <div class="usage-row" v-if="usage">
            <div><b>{{ fmt(usage.total_words) }}</b><span>累计字数</span></div>
            <div><b>{{ fmt(usage.month_words) }}</b><span>本月</span></div>
            <div><b>{{ fmt(usage.today_words) }}</b><span>今日</span></div>
          </div>
          <button class="btn primary" @click="saveProfile" :disabled="saving">{{ saving ? '保存中…' : '保存' }}</button>

          <!-- 改密码（默认收起） -->
          <div class="pwd-toggle" @click="pwdOpen = !pwdOpen">修改密码 {{ pwdOpen ? '▲' : '▼' }}</div>
          <template v-if="pwdOpen">
            <label class="field"><span>旧密码</span>
              <input v-model="pwd.old" type="password" class="input" placeholder="当前密码" /></label>
            <label class="field"><span>新密码</span>
              <input v-model="pwd.next" type="password" class="input" placeholder="至少 6 位" /></label>
            <label class="field"><span>确认新密码</span>
              <input v-model="pwd.confirm" type="password" class="input" placeholder="再输一遍" @keyup.enter="changePwd" /></label>
            <button class="btn" @click="changePwd" :disabled="changingPwd">{{ changingPwd ? '提交中…' : '修改密码' }}</button>
          </template>

          <button class="btn logout-btn" @click="doLogout">退出登录</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'
import { toast } from '../composables/useToast'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()

const router = useRouter()
const EMOJIS = ['✍️', '📖', '🐉', '🗡️', '🌸', '🌙', '⭐', '🔥', '🦊', '🐱', '🐼', '🧙']
const PLAN_NAMES: Record<string, string> = { free: '免费版', vip: 'VIP', svip: 'SVIP' }

const me = ref<any>({})
const form = ref({ nickname: '', avatar: '' })
const pwd = ref({ old: '', next: '', confirm: '' })
const pwdOpen = ref(false)
const usage = ref<any>(null)
const saving = ref(false)
const changingPwd = ref(false)

const initials = computed(() => {
  if (form.value.avatar) return ''
  const n = form.value.nickname || me.value.email || '?'
  return n.slice(0, 1).toUpperCase()
})
const avatarStyle = computed(() => form.value.avatar
  ? { background: 'var(--bg-tertiary, #f0f0f3)', fontSize: '26px', lineHeight: '52px' }
  : { background: 'var(--accent, #007aff)', color: '#fff', fontSize: '22px', fontWeight: 700, lineHeight: '52px' })

function fmt(n: number) {
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return String(n ?? 0)
}

// 每次打开时拉最新
watch(() => props.open, async o => {
  if (!o) return
  pwdOpen.value = false
  pwd.value = { old: '', next: '', confirm: '' }
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
    toast.success('已保存')
    me.value.nickname = form.value.nickname
    me.value.avatar = form.value.avatar
    try {
      localStorage.setItem('xzj-nickname', form.value.nickname)
      localStorage.setItem('xzj-avatar', form.value.avatar)
    } catch {}
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
    pwdOpen.value = false
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

function doLogout() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  try { localStorage.removeItem('xzj-nickname'); localStorage.removeItem('xzj-avatar') } catch {}
  emit('update:open', false)
  router.push('/login')
}
</script>

<style scoped>
.pm-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.4); z-index: 200;
  display: flex; align-items: center; justify-content: center; padding: 16px;
}
.pm-modal {
  width: 380px; max-width: 100%; max-height: 88vh; overflow-y: auto;
  background: var(--bg-secondary, #fff); border-radius: 16px;
  box-shadow: 0 16px 48px rgba(0,0,0,.22);
}
.pm-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 18px; border-bottom: 1px solid var(--border, #eee); font-weight: 600;
}
.pm-close { border: none; background: none; cursor: pointer; font-size: 15px; color: var(--text-tertiary); }
.pm-body { padding: 16px 18px 18px; }
.avatar-row { display: flex; gap: 14px; align-items: center; margin-bottom: 12px; }
.avatar { width: 52px; height: 52px; border-radius: 50%; text-align: center; flex-shrink: 0; overflow: hidden; }
.avatar-ops { flex: 1; min-width: 0; }
.emoji-picker { display: flex; flex-wrap: wrap; gap: 5px; }
.emoji-btn {
  width: 32px; height: 32px; border-radius: 8px; border: 1px solid var(--border, #e5e5ea);
  background: var(--bg-secondary, #fff); font-size: 16px; cursor: pointer; padding: 0;
}
.emoji-btn.on { border-color: var(--accent, #007aff); background: rgba(0,122,255,.08); }
.emoji-clear { font-size: 12px; color: var(--text-tertiary); }
.field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 10px; }
.field span { font-size: 12px; color: var(--text-secondary); }
.input {
  padding: 8px 11px; border: 1px solid var(--border, #e5e5ea); border-radius: 8px;
  font-size: 14px; outline: none; background: var(--bg-secondary, #fff); color: var(--text-primary);
}
.input:focus { border-color: var(--accent, #007aff); }
.meta-row { display: flex; justify-content: space-between; font-size: 13px; padding: 4px 0; }
.meta-row span { color: var(--text-secondary); }
.meta-row b { color: var(--text-primary); font-weight: 500; }
.usage-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin: 10px 0 12px; }
.usage-row > div { background: var(--bg-tertiary, #f5f5f7); border-radius: 10px; padding: 9px 4px; text-align: center; }
.usage-row b { display: block; font-size: 15px; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.usage-row span { font-size: 10px; color: var(--text-tertiary); }
.btn {
  width: 100%; border: 1px solid var(--border, #e0e0e0); background: transparent; padding: 9px 0;
  border-radius: 9px; cursor: pointer; font-size: 14px; color: var(--text-primary); margin-top: 8px;
}
.btn.primary { background: var(--accent, #007aff); color: #fff; border-color: var(--accent, #007aff); }
.btn.primary:disabled, .btn:disabled { opacity: .6; cursor: not-allowed; }
.pwd-toggle {
  font-size: 13px; color: var(--text-secondary); text-align: center; cursor: pointer;
  padding: 10px 0 6px; margin-top: 6px; border-top: 1px solid var(--border, #f0f0f3);
}
.pwd-toggle:hover { color: var(--text-primary); }
.logout-btn { color: #d70015; border-color: rgba(255,59,48,.3); }
.logout-btn:hover { background: rgba(255,59,48,.06); }
</style>
