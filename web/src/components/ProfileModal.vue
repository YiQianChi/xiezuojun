<template>
  <!-- 下拉菜单（导航栏头像点开） -->
  <Teleport to="body">
    <div v-if="menuOpen" class="pmenu-overlay" @click="closeMenu"></div>
    <div v-if="menuOpen" class="pmenu" :style="menuPos" @click.stop>
      <button class="pmenu-item" @click="openEdit">修改信息</button>
      <button class="pmenu-item" @click="openPwd">修改密码</button>
      <button class="pmenu-item" @click="openSecurity">账号与安全</button>
      <div class="pmenu-sep"></div>
      <button class="pmenu-item danger" @click="doLogout">退出登录</button>
    </div>

    <!-- 修改信息/起笔名 弹窗（极简版：大头像+笔名，头像后续接上传） -->
    <div v-if="editOpen" class="pdlg-overlay" @click.self="editOpen = false">
      <div class="pdlg">
        <div class="pdlg-head"><span>{{ penNameMode ? '起个笔名' : '修改信息' }}</span><button @click="editOpen = false">✕</button></div>
        <div class="pdlg-body">
          <!-- 大头像（居中，点击=随机换 emoji；后续这里接上传） -->
          <div class="big-avatar-wrap">
            <div class="big-avatar" :style="avatarStyle" @click="cycleAvatar" :title="'点头像换一个'">{{ form.avatar || initials }}</div>
            <span class="big-avatar-tip">{{ form.avatar ? '点头像换' : '点头像选一个' }}</span>
          </div>
          <p v-if="penNameMode" class="pen-hint">写作菌里大家用笔名相称——搭子配对时对方只能看到你的笔名，邮箱不会展示。</p>
          <label class="field">
            <span>笔名</span>
            <input v-model="form.nickname" class="input" placeholder="给自己起个笔名" maxlength="20" @keyup.enter="saveProfile" />
          </label>
          <div class="pen-ops">
            <button class="btn reroll" @click="reroll">🎲 笔名</button>
            <button class="btn primary" @click="saveProfile" :disabled="saving">{{ saving ? '保存中…' : (penNameMode ? '就用这个' : '保存') }}</button>
          </div>
          <button v-if="penNameMode" class="btn skip" @click="editOpen = false">先跳过</button>
          <div v-else class="meta-row"><span>邮箱</span><b>{{ me.email }}</b></div>
        </div>
      </div>
    </div>

    <!-- 修改密码弹窗（无密码用户=首次设置密码，免旧密码；无邮箱用户引导去设置邮箱——密码登录需要邮箱做账号） -->
    <div v-if="pwdOpen" class="pdlg-overlay" @click.self="pwdOpen = false">
      <div class="pdlg">
        <div class="pdlg-head"><span>{{ pwdHas ? '修改密码' : '设置密码' }}</span><button @click="pwdOpen = false">✕</button></div>
        <div class="pdlg-body">
          <p v-if="!pwdHas" class="pwd-hint">你还没有密码（微信/手机号登录建的号）。设置后可用邮箱 + 密码登录电脑版。</p>
          <p v-if="!pwdHas && pwdNoEmail" class="pwd-hint">提示：密码登录还需要邮箱做账号，建议之后在「账号与安全」里绑定邮箱。</p>
          <label v-if="pwdHas" class="field"><span>旧密码</span>
            <input v-model="pwd.old" type="password" class="input" placeholder="当前密码" /></label>
          <label class="field"><span>{{ pwdHas ? '新密码' : '密码' }}</span>
            <input v-model="pwd.next" type="password" class="input" placeholder="至少 6 位" /></label>
          <label class="field"><span>确认密码</span>
            <input v-model="pwd.confirm" type="password" class="input" placeholder="再输一遍" @keyup.enter="changePwd" /></label>
          <button class="btn primary" @click="changePwd" :disabled="changingPwd">{{ changingPwd ? '提交中…' : (pwdHas ? '修改密码' : '设置密码') }}</button>
        </div>
      </div>
    </div>

    <!-- 账号与安全：绑定状态一览 + 补绑 -->
    <div v-if="secOpen" class="pdlg-overlay" @click.self="secOpen = false">
      <div class="pdlg sec-dlg">
        <div class="pdlg-head"><span>账号与安全</span><button @click="secOpen = false">✕</button></div>
        <div class="pdlg-body">
          <p class="sec-desc">以下方式都可登录同一个账号，建议至少保留两种</p>

          <div class="sec-card" :class="{ bound: sec.email_bound }">
            <div class="sec-ico">✉️</div>
            <div class="sec-info">
              <b>邮箱</b>
              <span>{{ sec.email_bound ? (me.email || '已绑定') : '未绑定，可补一个用于登录和找回' }}</span>
            </div>
            <span v-if="sec.email_bound" class="sec-badge">已绑定</span>
            <button v-else class="btn sec-op" @click="emailOpen = !emailOpen">{{ emailOpen ? '收起' : '设置' }}</button>
          </div>
          <div v-if="!sec.email_bound && emailOpen" class="sec-form">
            <input v-model.trim="secEmail" type="email" class="input" placeholder="邮箱地址（未注册过的）" />
            <div class="sec-code">
              <input v-model.trim="secEmailCode" inputmode="numeric" maxlength="6" class="input" placeholder="邮箱验证码" @keyup.enter="doSetEmail" />
              <button class="btn sec-op" :disabled="emailSending || emailCountdown > 0" @click="sendEmailBindCode">
                {{ emailCountdown > 0 ? emailCountdown + 's' : '获取验证码' }}
              </button>
            </div>
            <button class="btn primary sec-save" @click="doSetEmail" :disabled="secSaving">绑定邮箱</button>
            <p class="pwd-hint">已被注册的邮箱不能绑定，请换一个。密码请在「修改密码」里单独设置。</p>
          </div>

          <div class="sec-card" :class="{ bound: sec.phone_bound }">
            <div class="sec-ico">📱</div>
            <div class="sec-info">
              <b>手机号</b>
              <span>{{ sec.phone_bound ? (me.phone || '已绑定') : '未绑定，验证短信后可手机号登录' }}</span>
            </div>
            <span v-if="sec.phone_bound" class="sec-badge">已绑定</span>
            <button v-else class="btn sec-op" @click="phoneOpen = !phoneOpen">{{ phoneOpen ? '收起' : '绑定' }}</button>
          </div>
          <div v-if="!sec.phone_bound && phoneOpen" class="sec-form">
            <div class="sec-code">
              <input v-model.trim="bindPhoneNo" inputmode="numeric" maxlength="11" class="input" placeholder="手机号" />
              <button class="btn sec-op" :disabled="bindSending || bindCountdown > 0" @click="sendBindCode">
                {{ bindCountdown > 0 ? bindCountdown + 's' : '获取验证码' }}
              </button>
            </div>
            <input v-model.trim="bindCode" inputmode="numeric" maxlength="6" class="input" placeholder="短信验证码" @keyup.enter="doBindPhone" />
            <button class="btn primary sec-save" @click="doBindPhone" :disabled="secSaving">绑定</button>
          </div>

          <div class="sec-card" :class="{ bound: sec.wechat_bound }">
            <div class="sec-ico">💬</div>
            <div class="sec-info">
              <b>微信</b>
              <span>{{ sec.wechat_bound ? '扫码 / 小程序登录同一账号' : '未绑定，扫码后微信登录即本账号' }}</span>
            </div>
            <span v-if="sec.wechat_bound" class="sec-badge">已绑定</span>
            <button v-else class="btn sec-op" @click="doWxBind">扫码绑定</button>
          </div>

          <!-- 统一身份冲突处理：微信/手机被占 → 验证改绑；孤儿号 → 并入当前账号 -->
          <div v-if="conflict" class="sec-form" style="margin-top:10px">
            <p class="sec-merge-hint">
              该{{ conflict.provider === 'phone' ? '手机号' : '微信' }}已绑定账号（{{ conflict.other_hint }}）。
              <template v-if="!conflict.orphan">输入对方账号收到的验证码，即可改绑到当前账号（对方账号保留其他登录方式）。</template>
              <template v-else>该账号只有这一个登录方式——可把它并入当前账号，作品会随迁。</template>
            </p>
            <template v-if="!conflict.orphan">
              <div class="sec-code">
                <input v-model.trim="conflictCode" inputmode="numeric" maxlength="6" class="input" placeholder="验证码" @keyup.enter="resolveConflict" />
                <button class="btn sec-op" :disabled="conflictSending || conflictCountdown > 0" @click="sendConflictCode">
                  {{ conflictCountdown > 0 ? conflictCountdown + 's' : '获取验证码' }}
                </button>
              </div>
              <button class="btn primary sec-save" @click="resolveConflict" :disabled="conflictSaving">确认改绑</button>
            </template>
            <button v-else class="btn primary sec-save" @click="mergeConflict" :disabled="conflictSaving">并入当前账号</button>
            <button class="btn sec-op" @click="conflict = null">取消</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 微信扫码绑定浮层 -->
    <div v-if="wxBindQr" class="pdlg-overlay">
      <div class="pdlg" style="text-align:center">
        <div class="pdlg-head"><span>微信扫码绑定</span><button @click="wxBindQr = false">✕</button></div>
        <div class="pdlg-body">
          <iframe class="wxbind-frame" :src="wxBindSrc" frameborder="0" scrolling="no"></iframe>
          <p class="bind-hint">用微信扫一扫，确认后自动绑定到当前账号</p>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { userApi } from '../core/api'
import { toast } from '../composables/useToast'
import { mpNavigate, isMiniProgramEnv } from '../core/mp'

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
defineExpose({ openMenu, openForPenName, openSecurityEmail, openPwd })

// 外部入口：打开账号与安全并展开设置邮箱表单（App 全局提示「设置密码」用）
async function openSecurityEmail() {
  emailOpen.value = true
  await openSecurity()
}

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
  ? { background: 'var(--bg-tertiary, #f0f0f3)', fontSize: '34px', lineHeight: '64px' }
  : { background: 'var(--accent, #007aff)', color: '#fff', fontSize: '28px', fontWeight: 700, lineHeight: '66px' })

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

// 点大头像循环换 emoji（简单方案；后续接用户上传后此入口换成"上传/换头像"）
function cycleAvatar() {
  const cur = form.value.avatar
  const idx = cur ? EMOJIS.indexOf(cur) : -1
  form.value.avatar = EMOJIS[(idx + 1) % EMOJIS.length]
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

async function openPwd() {
  closeMenu()
  pwd.value = { old: '', next: '', confirm: '' }
  // 有没有密码决定弹窗形态（修改 vs 首次设置）；没邮箱不拦截——只是提示密码登录需要邮箱
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
  } catch {}
  pwdOpen.value = true
}

const pwdHas = computed(() => !!(me.value.security?.has_password))
const pwdNoEmail = computed(() => !me.value.security?.email_bound)

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
  if (pwdHas.value && !o) { toast.error('请填写完整'); return }
  if (!n) { toast.error('请填写新密码'); return }
  if (n.length < 6) { toast.error('新密码至少 6 位'); return }
  if (n !== c) { toast.error('两次新密码不一致'); return }
  changingPwd.value = true
  try {
    await userApi.changePassword(pwdHas.value ? { old_password: o, new_password: n } : { new_password: n })
    toast.success(pwdHas.value ? '密码已修改' : '密码已设置')
    pwdOpen.value = false
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '修改失败')
  } finally {
    changingPwd.value = false
  }
}

// ===== 账号与安全 =====
const secOpen = ref(false)
const sec = ref<any>({})
const emailOpen = ref(false)
const phoneOpen = ref(false)
// 展开/收起时重置合并模式（换邮箱后旧状态不留）
const secEmail = ref('')
const secEmailCode = ref('')
const emailSending = ref(false)
const emailCountdown = ref(0)

// 统一身份冲突（微信/手机被占）
const conflict = ref<any>(null)
const conflictCode = ref('')
const conflictSending = ref(false)
const conflictSaving = ref(false)
const conflictCountdown = ref(0)
const bindPhoneNo = ref('')
const bindCode = ref('')
const bindSending = ref(false)
const bindCountdown = ref(0)
const secSaving = ref(false)
const wxBindQr = ref(false)
const wxBindSrc = ref('')

async function openSecurity() {
  closeMenu()
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
    sec.value = me.value.security || {}
  } catch {}
  secOpen.value = true
}

async function refreshSec() {
  try {
    const res = await userApi.me()
    me.value = res.data?.data || {}
    sec.value = me.value.security || {}
  } catch {}
}

async function sendEmailBindCode() {
  if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(secEmail.value)) { toast.error('请输入正确的邮箱'); return }
  emailSending.value = true
  try {
    await userApi.sendEmailCode({ email: secEmail.value, scene: 'bind' })
    toast.success('验证码已发送')
    emailCountdown.value = 60
    const t = window.setInterval(() => { emailCountdown.value -= 1; if (emailCountdown.value <= 0) window.clearInterval(t) }, 1000)
  } catch (e: any) {
    // 已注册的邮箱不能绑定（不合并），后端 409 文案直接展示引导换邮箱
    toast.error(e?.response?.data?.error || '发送失败')
  } finally { emailSending.value = false }
}

// ===== 统一身份冲突处理 =====
async function createPhoneConflict(phone: string) {
  try {
    const res = await userApi.identityConflict({ provider: 'phone', identifier: phone })
    conflict.value = res.data?.data || res.data
    conflictCode.value = ''
    toast.info('该手机号已绑定其他账号——验证后可改绑到当前账号')
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '无法处理该冲突')
  }
}

async function sendConflictCode() {
  if (!conflict.value?.ticket) return
  conflictSending.value = true
  try {
    const res = await userApi.idConflictSendCode(conflict.value.ticket)
    toast.success('验证码已发到 ' + ((res.data?.data || {}).sent_to || '对方账号'))
    conflictCountdown.value = 60
    const t = window.setInterval(() => { conflictCountdown.value -= 1; if (conflictCountdown.value <= 0) window.clearInterval(t) }, 1000)
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '发送失败')
  } finally { conflictSending.value = false }
}

async function resolveConflict() {
  if (!/^\d{6}$/.test(conflictCode.value)) { toast.error('请输入 6 位验证码'); return }
  conflictSaving.value = true
  try {
    await userApi.idConflictResolve({ ticket: conflict.value.ticket, code: conflictCode.value })
    toast.success('已改绑到当前账号')
    conflict.value = null
    await refreshSec()
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '改绑失败')
  } finally { conflictSaving.value = false }
}

async function mergeConflict() {
  conflictSaving.value = true
  try {
    await userApi.idConflictMerge(conflict.value.ticket)
    toast.success('已并入当前账号，作品随迁完成')
    conflict.value = null
    await refreshSec()
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '合并失败')
  } finally { conflictSaving.value = false }
}

async function doSetEmail() {
  if (!secEmail.value) { toast.error('请填邮箱'); return }
  if (!/^\d{6}$/.test(secEmailCode.value)) { toast.error('请输入 6 位邮箱验证码'); return }
  secSaving.value = true
  try {
    await userApi.setEmail({ email: secEmail.value, code: secEmailCode.value })
    toast.success('邮箱已设置')
    emailOpen.value = false
    await refreshSec()
    window.dispatchEvent(new CustomEvent('xzj-profile-updated'))
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '设置失败')
  } finally { secSaving.value = false }
}

async function sendBindCode() {
  if (!/^1[3-9]\d{9}$/.test(bindPhoneNo.value)) { toast.error('请输入正确的手机号'); return }
  bindSending.value = true
  try {
    const res = await userApi.sendSmsCode({ phone: bindPhoneNo.value, scene: 'bind' })
    const c = res.data?.data?.code
    if (c) toast.success('测试码：' + c)
    else toast.success('验证码已发送')
    bindCountdown.value = 60
    const t = window.setInterval(() => { bindCountdown.value -= 1; if (bindCountdown.value <= 0) window.clearInterval(t) }, 1000)
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '发送失败')
  } finally { bindSending.value = false }
}

async function doBindPhone() {
  if (!/^\d{6}$/.test(bindCode.value)) { toast.error('请输入 6 位验证码'); return }
  secSaving.value = true
  try {
    await userApi.bindPhone({ phone: bindPhoneNo.value, code: bindCode.value })
    toast.success('手机号已绑定')
    phoneOpen.value = false
    await refreshSec()
  } catch (e: any) {
    if (e?.response?.status === 409) {
      // 被占：转统一冲突处理（验证改绑 / 孤儿号合并）
      await createPhoneConflict(bindPhoneNo.value)
      return
    }
    toast.error(e?.response?.data?.error || '绑定失败')
  } finally { secSaving.value = false }
}

async function doWxBind() {
  try {
    const res = await userApi.wxBindTicket()
    const t = res.data?.data?.ticket
    if (!t) { toast.error('获取绑定票据失败'); return }
    wxBindSrc.value = '/api/auth/wx-oauth/start?embed=1&bind_ticket=' + t +
      '&redirect=' + encodeURIComponent(window.location.origin + window.location.pathname)
    wxBindQr.value = true
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '发起绑定失败')
  }
}

const onWxBindMsg = async (e: MessageEvent) => {
  if (e.origin !== window.location.origin || !e.data || e.data.type !== 'xzj-wx-bind') return
  wxBindQr.value = false
  if (e.data.ok) {
    toast.success('微信绑定成功')
    await refreshSec()
  } else if (e.data.conflict) {
    // 被占：后端已建冲突票据，转统一冲突处理面板
    conflict.value = e.data.conflict
    conflictCode.value = ''
    toast.info('该微信已绑定其他账号——按提示完成验证或合并')
  } else {
    toast.error(e.data.error || '绑定失败')
  }
}

onMounted(() => window.addEventListener('message', onWxBindMsg))
onUnmounted(() => window.removeEventListener('message', onWxBindMsg))

function doLogout() {
  closeMenu()
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  try { localStorage.removeItem('xzj-nickname'); localStorage.removeItem('xzj-avatar') } catch {}
  // 小程序：回壳的原生登录页（壳会在该页清掉 storage，否则点左上角主页又会拿旧 token 自动登回来）
  if (isMiniProgramEnv) {
    mpNavigate('/pages/login/login')
    return
  }
  router.push('/login')
}
</script>

<style scoped>
/* 下拉菜单 */
.pmenu-overlay { position: fixed; inset: 0; z-index: 200; }
.pmenu {
  position: fixed; z-index: 201; min-width: 156px;
  background: var(--bg-secondary, #fff); border-radius: 14px;
  box-shadow: 0 16px 48px rgba(0,0,0,.16), 0 2px 8px rgba(0,0,0,.05); padding: 6px;
  border: 1px solid rgba(0,0,0,.04);
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
  width: 340px; max-width: 100%; max-height: 88dvh;
  display: flex; flex-direction: column;
  background: var(--bg-secondary, #fff); border-radius: 20px;
  box-shadow: 0 24px 64px rgba(0,0,0,.18), 0 2px 8px rgba(0,0,0,.06);
  overflow: hidden;
}
.pdlg-head { flex-shrink: 0; }
.pdlg-body { overflow-y: auto; -webkit-overflow-scrolling: touch; }
.pdlg-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 16px 6px; font-weight: 600; font-size: 15px; color: var(--text-primary);
}
.pdlg-head button {
  border: none; background: none; cursor: pointer; font-size: 14px; color: var(--text-tertiary);
  width: 28px; height: 28px; border-radius: 50%; line-height: 1;
}
.pdlg-head button:hover { background: var(--bg-tertiary, #f0f0f3); color: var(--text-secondary); }
.pdlg-body { padding: 14px 16px 16px; }

.big-avatar-wrap {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  margin: 4px -16px 14px; padding: 18px 0 14px;
  background: linear-gradient(180deg, var(--bg-tertiary, #f5f5f7), transparent);
}
.big-avatar {
  width: 72px; height: 72px; border-radius: 50%; text-align: center; overflow: hidden;
  cursor: pointer; user-select: none; transition: transform .15s ease, box-shadow .15s ease;
  box-shadow: 0 4px 16px rgba(0,122,255,.22);
  border: 3px solid #fff; outline: 1px solid var(--border, #e0e0e0);
}
.big-avatar:hover { transform: scale(1.06); box-shadow: 0 6px 22px rgba(0,122,255,.3); }
.big-avatar:active { transform: scale(.97); }
.big-avatar-tip { font-size: 11px; color: var(--text-tertiary); }
.field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 10px; }
.field span { font-size: 12px; color: var(--text-secondary); }
.input {
  padding: 9px 12px; border: 1.5px solid var(--border, #e5e5ea); border-radius: 10px;
  font-size: 14px; outline: none; background: var(--bg-secondary, #fff); color: var(--text-primary);
  transition: border-color .15s, box-shadow .15s;
}
.input:focus { border-color: var(--accent, #007aff); box-shadow: 0 0 0 3px rgba(0,122,255,.12); }
.meta-row { display: flex; justify-content: space-between; font-size: 13px; padding: 4px 0; }
.meta-row span { color: var(--text-secondary); }
.meta-row b { color: var(--text-primary); font-weight: 500; }
.btn {
  width: 100%; border: 1px solid var(--border, #e0e0e0); background: transparent; padding: 10px 0;
  border-radius: 11px; cursor: pointer; font-size: 14px; color: var(--text-primary); margin-top: 8px;
  transition: transform .1s ease, box-shadow .15s ease, opacity .15s;
}
.btn.primary {
  background: var(--accent, #007aff); color: #fff; border: none; font-weight: 500;
  box-shadow: 0 3px 12px rgba(0,122,255,.25);
}
.btn.primary:hover { box-shadow: 0 5px 18px rgba(0,122,255,.35); }
.btn.primary:active { transform: scale(.98); }
.btn:disabled { opacity: .55; cursor: not-allowed; }
.pen-hint { font-size: 12px; color: var(--text-secondary); line-height: 1.6; margin: 0 0 10px; background: var(--bg-tertiary, #f5f5f7); border-radius: 10px; padding: 9px 11px; }
.pen-ops { display: flex; gap: 8px; margin-top: 8px; }
.pen-ops .btn { flex: 1; width: auto; margin-top: 0; min-width: 0; }
.pen-ops .reroll {
  flex: 0 0 auto; width: auto; padding: 10px 16px; margin-top: 0;
  background: var(--bg-tertiary, #f0f0f3); border: none; border-radius: 11px;
  color: var(--text-secondary); font-size: 14px; font-weight: 500;
  transition: background .15s, transform .1s;
}
.pen-ops .reroll:hover { background: var(--border, #e0e0e0); }
.pen-ops .reroll:active { transform: scale(.96); }
.btn.skip { margin-top: 10px; border: none; color: var(--text-tertiary); font-size: 13px; padding: 7px 0; }
.btn.skip:hover { color: var(--text-secondary); background: var(--bg-tertiary, #f5f5f7); }
.bind-hint { font-size: 12px; color: var(--text-tertiary, #999); line-height: 1.6; margin: 0 0 12px; }
.sec-dlg { width: 400px; }
.sec-desc { font-size:12px; color:var(--text-tertiary,#999); margin:0 0 12px; }
.sec-card {
  display:flex; align-items:center; gap:12px; padding:12px 14px; margin-bottom:10px;
  background:var(--bg-tertiary,#f7f7fa); border-radius:14px;
  border:1px solid transparent; transition:border-color .15s;
}
.sec-card.bound { border-color:rgba(52,199,89,.25); }
.sec-ico { width:38px; height:38px; border-radius:12px; background:var(--bg-secondary,#fff);
  display:flex; align-items:center; justify-content:center; font-size:19px; flex-shrink:0;
  box-shadow:0 1px 4px rgba(0,0,0,.06); }
.sec-info { flex:1; min-width:0; }
.sec-info b { display:block; font-size:14px; color:var(--text-primary); margin-bottom:2px; }
.sec-info span { display:block; font-size:12px; color:var(--text-secondary);
  overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.sec-badge {
  font-size:11px; font-weight:600; color:#248a3d; background:rgba(52,199,89,.12);
  padding:3px 10px; border-radius:99px; white-space:nowrap;
}
.sec-op { padding:6px 14px; font-size:12px; white-space:nowrap; border-radius:10px;
  width:auto !important; flex:0 0 auto; align-self:center; margin-top:0; }
.sec-form {
  display:flex; flex-direction:column; gap:8px; padding:12px 14px 14px; margin:-4px 0 10px;
  background:var(--bg-tertiary,#f7f7fa); border-radius:14px;
}
.sec-code { display:flex; gap:8px; }
.pwd-hint { font-size: 12px; color: var(--text-secondary); line-height: 1.6; margin: 0 0 12px; }
.sec-merge-hint { font-size: 12px; color: var(--text-secondary); line-height: 1.7; margin: 0 0 12px; }
.sec-form .sec-op { margin-top: 8px; }
.sec-code .input { flex:1; }
.sec-save { margin-top:2px; }
/* 手机：弹窗满宽、按钮加高到手指友好 */
@media (max-width: 480px) {
  .sec-dlg { width: 100%; }
  .sec-op { padding: 8px 16px; }
  .sec-ico { width: 34px; height: 34px; font-size: 17px; }
  .sec-card { padding: 10px 12px; }
}
.wxbind-frame { width:300px; height:340px; border:none; margin:0 auto; }
</style>
