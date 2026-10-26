<template>
  <!-- 下拉菜单（导航栏头像点开） -->
  <Teleport to="body">
    <div v-if="menuOpen" class="pmenu-overlay" @click="closeMenu"></div>
    <div v-if="menuOpen" class="pmenu" :style="menuPos" @click.stop>
      <button class="pmenu-item" @click="openEdit">修改信息</button>
      <button class="pmenu-item" @click="openPwd">修改密码</button>
      <div class="pmenu-sep"></div>
      <button class="pmenu-item danger" @click="doLogout">退出登录</button>
    </div>

    <!-- 修改信息/起笔名 弹窗 -->
    <div v-if="editOpen" class="pdlg-overlay" @click.self="editOpen = false">
      <div class="pdlg">
        <div class="pdlg-head"><span>{{ penNameMode ? '起个笔名' : '修改信息' }}</span><button @click="editOpen = false">✕</button></div>
        <div class="pdlg-body">
          <div class="avatar-row">
            <div class="avatar" :style="avatarStyle">{{ form.avatar || initials }}</div>
            <div class="avatar-ops">
              <div class="emoji-picker">
                <button v-for="e in EMOJIS" :key="e" class="emoji-btn" :class="{ on: form.avatar === e }" @click="form.avatar = e">{{ e }}</button>
                <button v-if="form.avatar" class="emoji-btn emoji-clear" @click="form.avatar = ''">✕</button>
              </div>
            </div>
          </div>
          <p v-if="penNameMode" class="pen-hint">写作菌里大家用笔名相称——搭子配对时对方只能看到你的笔名，邮箱等隐私不会展示。已为你随机生成一个，可以直接用或改成自己的。</p>
          <label class="field">
            <span>笔名</span>
            <input v-model="form.nickname" class="input" placeholder="给自己起个笔名" maxlength="20" @keyup.enter="saveProfile" />
          </label>
          <div class="pen-ops">
            <button class="btn reroll" @click="reroll">🎲 换一个</button>
            <button class="btn primary" @click="saveProfile" :disabled="saving">{{ saving ? '保存中…' : (penNameMode ? '就用这个' : '保存') }}</button>
          </div>
          <button v-if="penNameMode" class="btn skip" @click="editOpen = false">先跳过</button>
          <div v-else class="meta-row"><span>邮箱</span><b>{{ me.email }}</b></div>
        </div>
      </div>
    </div>

    <!-- 修改密码弹窗 -->
    <div v-if="pwdOpen" class="pdlg-overlay" @click.self="pwdOpen = false">
      <div class="pdlg">
        <div class="pdlg-head"><span>修改密码</span><button @click="pwdOpen = false">✕</button></div>
        <div class="pdlg-body">
          <label class="field"><span>旧密码</span>
            <input v-model="pwd.old" type="password" class="input" placeholder="当前密码" /></label>
          <label class="field"><span>新密码</span>
            <input v-model="pwd.next" type="password" class="input" placeholder="至少 6 位" /></label>
          <label class="field"><span>确认新密码</span>
            <input v-model="pwd.confirm" type="password" class="input" placeholder="再输一遍" @keyup.enter="changePwd" /></label>
          <button class="btn primary" @click="changePwd" :disabled="changingPwd">{{ changingPwd ? '提交中…' : '修改密码' }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const EMOJIS = ['✍️', '📖', '🐉', '🗡️', '🌸', '🌙', '⭐', '🔥', '🦊', '🐱', '🐼', '🧙']

// 菜单定位（头像按钮正下方）
const menuOpen = ref(false)
const menuPos = ref({ top: '0px', right: '12px' })
function openMenu(anchor: HTMLElement) {
  const r = anchor.getBoundingClientRect()
  menuPos.value = { top: r.bottom + 8 + 'px', right: (window.innerWidth - r.right) + 'px' }
  menuOpen.value = true
}
function closeMenu() { menuOpen.value = false }
defineExpose({ openMenu, openForPenName })

const editOpen = ref(false)
const penNameMode = ref(false)  // 首次起笔名模式（needs_pen_name 触发）
const pwdOpen = ref(false)
const me = ref<any>({})
const form = ref({ nickname: '', avatar: '' })
const pwd = ref({ old: '', next: '', confirm: '' })
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

// 注册后首启：拉到 needs_pen_name=true 时调用（由 App.vue 触发）
async function openForPenName() {
  penNameMode.value = true
  editOpen.value = true
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
    form.value.nickname = me.value.nickname || ''
    form.value.avatar = me.value.avatar || ''
  } catch {}
}

function reroll() {
  const adjs = ['墨水','熬夜','故事','灵感','断更','催更','柠檬','月亮','布谷','青苔','拾光','听雨','衔月','踏雪','执笔','小饿','打盹','云朵','萤火','薄荷']
  const nouns = ['菌']
  form.value.nickname = adjs[Math.floor(Math.random()*adjs.length)] + nouns[Math.floor(Math.random()*nouns.length)] + String(Math.floor(Math.random()*90)+10)
}

async function openEdit() {
  closeMenu()
  penNameMode.value = false
  editOpen.value = true
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
    form.value.nickname = me.value.nickname || ''
    form.value.avatar = me.value.avatar || ''
  } catch {}
}

function openPwd() {
  closeMenu()
  pwd.value = { old: '', next: '', confirm: '' }
  pwdOpen.value = true
}

async function saveProfile() {
  saving.value = true
  try {
    const payload: any = {}
    if (form.value.nickname !== (me.value.nickname || '')) payload.nickname = form.value.nickname
    if (form.value.avatar !== (me.value.avatar || '')) payload.avatar = form.value.avatar
    if (!Object.keys(payload).length) { editOpen.value = false; return }
    await userApi.updateProfile(payload)
    toast.success('已保存')
    me.value.nickname = form.value.nickname
    me.value.avatar = form.value.avatar
    try {
      localStorage.setItem('xzj-nickname', form.value.nickname)
      localStorage.setItem('xzj-avatar', form.value.avatar)
    } catch {}
    window.dispatchEvent(new CustomEvent('xzj-profile-updated'))
    editOpen.value = false
    penNameMode.value = false
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
    pwdOpen.value = false
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

function doLogout() {
  closeMenu()
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  try { localStorage.removeItem('xzj-nickname'); localStorage.removeItem('xzj-avatar') } catch {}
  router.push('/login')
}
</script>

<style scoped>
/* 下拉菜单 */
.pmenu-overlay { position: fixed; inset: 0; z-index: 200; }
.pmenu {
  position: fixed; z-index: 201; min-width: 150px;
  background: var(--bg-secondary, #fff); border-radius: 12px;
  box-shadow: 0 8px 30px rgba(0,0,0,.16); padding: 5px;
}
.pmenu-item {
  display: block; width: 100%; text-align: left; border: none; background: none;
  padding: 9px 14px; font-size: 14px; color: var(--text-primary);
  border-radius: 8px; cursor: pointer;
}
.pmenu-item:hover { background: var(--bg-tertiary, #f5f5f7); }
.pmenu-item.danger { color: #d70015; }
.pmenu-sep { height: 1px; background: var(--border, #f0f0f3); margin: 4px 8px; }

/* 通用小弹窗 */
.pdlg-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.4); z-index: 210;
  display: flex; align-items: center; justify-content: center; padding: 16px;
}
.pdlg {
  width: 340px; max-width: 100%; max-height: 88vh; overflow-y: auto;
  background: var(--bg-secondary, #fff); border-radius: 16px;
  box-shadow: 0 16px 48px rgba(0,0,0,.22);
}
.pdlg-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 13px 16px; border-bottom: 1px solid var(--border, #eee); font-weight: 600; font-size: 15px;
}
.pdlg-head button { border: none; background: none; cursor: pointer; font-size: 15px; color: var(--text-tertiary); }
.pdlg-body { padding: 14px 16px 16px; }

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
.btn {
  width: 100%; border: 1px solid var(--border, #e0e0e0); background: transparent; padding: 9px 0;
  border-radius: 9px; cursor: pointer; font-size: 14px; color: var(--text-primary); margin-top: 8px;
}
.btn.primary { background: var(--accent, #007aff); color: #fff; border-color: var(--accent, #007aff); }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.pen-hint { font-size: 12px; color: var(--text-secondary); line-height: 1.6; margin: 0 0 10px; background: var(--bg-tertiary, #f5f5f7); border-radius: 8px; padding: 8px 10px; }
.pen-ops { display: flex; gap: 8px; margin-top: 8px; }
.pen-ops .btn { flex: 1; margin-top: 0; }
.pen-ops .reroll { flex: 0 0 auto; }
.btn.skip { margin-top: 8px; border: none; color: var(--text-tertiary); font-size: 13px; }
</style>
