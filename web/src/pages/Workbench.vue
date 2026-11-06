<template>
  <div class="wb">
    <div class="wb-brand">
      <img class="wb-logo" src="/icons/icon-192.png" alt="写作菌" />
      <h1 class="wb-name">写作菌</h1>
      <p class="wb-slogan">让灵感不被打扰</p>
    </div>

    <p class="wb-quote"><span>{{ typed }}</span><span class="wb-caret"></span></p>

    <button class="wb-start" @click="continueWriting">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14.4 3.9 20.1 9.6"/><path d="M17.3 7 4.3 20 3 21l1-1.3L17 6.7"/></svg>
      开始写作
    </button>
    <p class="wb-start-sub">{{ recent.length ? `接着写《${recent[0].name}》` : '从第一句话开始你的故事' }}</p>

    <div class="wb-tiles">
      <button class="wb-tile" @click="router.push('/bookshelf')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4.5A2 2 0 0 1 6 2.5h13v16H6a2 2 0 0 0-2 2V4.5Z"/><path d="M4 20.5A2 2 0 0 1 6 18.5h13"/><path d="M8 7h7"/><path d="M8 11h5"/></svg>
        <span>我的书架</span>
      </button>
      <button class="wb-tile" @click="showCreate = true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="16" rx="3"/><path d="M12 8.5v7"/><path d="M8.5 12h7"/></svg>
        <span>新建作品</span>
      </button>
      <button class="wb-tile" @click="router.push('/bookshelf?import=1')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M4 19.5h16"/></svg>
        <span>导入书稿</span>
      </button>
    </div>

    <!-- 新建弹窗（只要书名） -->
    <div v-if="showCreate" class="wb-mask" @click.self="showCreate = false">
      <div class="wb-dlg">
        <h3>新建作品</h3>
        <input v-model="newName" class="input" placeholder="书名" maxlength="50" @keyup.enter="createProject" />
        <div class="wb-dlg-btns">
          <button class="btn" @click="showCreate = false">取消</button>
          <button class="btn primary" :disabled="creating || !newName.trim()" @click="createProject">{{ creating ? '创建中…' : '创建' }}</button>
        </div>
      </div>
    </div>
    <!-- 没密码的用户（微信/手机号建号）首进提示设密码，可跳过；邮箱绑定去「账号与安全」单独做 -->
    <div v-if="showSetPwd" class="wb-mask" @click.self="showSetPwd = false">
      <div class="wb-dlg">
        <h3>设置密码</h3>
        <p class="wb-dlg-hint">你现在用微信/手机号登录。设置密码后，手机号或邮箱 + 密码也能登录。</p>
        <input v-model="pwd1" type="password" class="input" placeholder="密码（至少 6 位）" />
        <input v-model="pwd2" type="password" class="input" placeholder="确认密码" @keyup.enter="doSetPwd" />
        <div class="wb-dlg-btns">
          <button class="btn" @click="skipSetPwd">跳过</button>
          <button class="btn primary" :disabled="pwdSaving" @click="doSetPwd">{{ pwdSaving ? '保存中…' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { projectApi, userApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const projects = ref<any[]>([])
const showCreate = ref(false)
const newName = ref('')
const creating = ref(false)
const showSetPwd = ref(false)
const pwd1 = ref('')
const pwd2 = ref('')
const pwdSaving = ref(false)

function skipSetPwd() {
  localStorage.setItem('xzj_skip_setpwd', '1')
  showSetPwd.value = false
}

async function doSetPwd() {
  if (pwd1.value.length < 6) { toast.error('密码至少 6 位'); return }
  if (pwd1.value !== pwd2.value) { toast.error('两次密码不一致'); return }
  pwdSaving.value = true
  try {
    await userApi.changePassword({ new_password: pwd1.value })
    toast.success('密码已设置')
    showSetPwd.value = false
  } catch (e: any) {
    toast.error(e?.response?.data?.error || '设置失败')
  } finally { pwdSaving.value = false }
}

const recent = computed(() =>
  [...projects.value].sort((a, b) => +new Date(b.updated_at) - +new Date(a.updated_at)).slice(0, 1)
)

// 每日金句：按天轮换，打字机逐字出（数据不落地，写死在前端）
const QUOTES = [
  '「夜还长，茶还热，故事才写到一半。」',
  '「写不出来的时候，先把上一段再读一遍。」',
  '「所有的初稿都是臭的，先写完再说。」',
  '「人物活了，情节自己会走。」',
  '「今天写五百字，也是没有断更的一天。」',
  '「别怕写坏，怕的是没写。」',
  '「灵感是等不来的，是写出来的。」'
]
const quote = QUOTES[Math.floor(Date.now() / 86400000) % QUOTES.length]
const typed = ref('')
let timer: number | undefined
onMounted(async () => {
  let i = 0
  timer = window.setInterval(() => {
    typed.value = quote.slice(0, i)
    if (i++ > quote.length) window.clearInterval(timer)
  }, 130)
  try {
    const res = await projectApi.list()
    projects.value = res.data?.data || res.data || []
  } catch { /* 401 由拦截器去登录 */ }
  // 没密码的用户（微信/手机号建号）→ 提示设置密码（点过跳过就不再弹；「修改密码」里随时可设）
  try {
    if (!localStorage.getItem('xzj_skip_setpwd')) {
      const me = await userApi.me()
      const sec = (me.data?.data || me.data)?.security || {}
      if (!sec.has_password) showSetPwd.value = true
    }
  } catch { /* 拿不到就不提示 */ }
})
onUnmounted(() => window.clearInterval(timer))

function openBook(p: any) {
  router.push(`/editor/${p.public_id || p.id}`)
}
function continueWriting() {
  if (recent.value.length) openBook(recent.value[0])
  else showCreate.value = true
}

async function createProject() {
  if (!newName.value.trim() || creating.value) return
  creating.value = true
  try {
    const res = await projectApi.create({ name: newName.value.trim(), description: '', setting_types: [] })
    const project = res.data?.data || res.data
    router.push(`/editor/${project.public_id || project.id}`)
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.wb { width: 100%; max-width: 640px; margin: 0 auto; padding: 36px 24px 48px; text-align: center; }
/* width:100% 是必须的：flex 父级里 auto 边距会让本容器按内容收缩（金句打字变宽→三个格子跟着变宽），锁死后与子内容解耦 */

.wb-logo { width: 76px; height: 76px; border-radius: 22px; box-shadow: 0 8px 24px rgba(0, 122, 255, .22); }
.wb-name { font-size: 24px; font-weight: 800; letter-spacing: 1px; margin-top: 14px; color: var(--text-primary); }
.wb-slogan { font-size: 12px; color: var(--text-secondary); margin-top: 4px; }

.wb-quote { min-height: 48px; margin: 24px auto 0; max-width: 280px; font-size: 14px; line-height: 1.7; color: var(--text-primary); }
/* min-height=两行正文高（14×1.7×2）：打字从 0 到换行全程零纵向顶动（手机适配） */
.wb-caret { display: inline-block; width: 2px; height: 15px; background: var(--accent); vertical-align: -2px; margin-left: 1px; animation: wb-blink 1s steps(1) infinite; }
@keyframes wb-blink { 50% { opacity: 0; } }

.wb-start { display: inline-flex; align-items: center; gap: 8px; margin-top: 18px; padding: 14px 46px; border: none; border-radius: 999px; background: var(--accent); color: #fff; font-size: 16px; font-weight: 700; cursor: pointer; box-shadow: 0 6px 18px rgba(0, 122, 255, .3); }
.wb-start:active { opacity: .85; }
.wb-start svg { width: 18px; height: 18px; }
.wb-start-sub { font-size: 11px; color: var(--text-secondary); margin-top: 9px; }

.wb-tiles { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-top: 30px; }
.wb-tile { background: var(--bg-secondary); border: none; border-radius: var(--radius-md); padding: 15px 6px 13px; display: flex; flex-direction: column; align-items: center; gap: 8px; cursor: pointer; }
.wb-tile:active { opacity: .7; }
.wb-tile svg { width: 21px; height: 21px; color: var(--accent); }
.wb-tile span { font-size: 12px; color: var(--text-primary); }

.wb-mask { position: fixed; inset: 0; background: rgba(0,0,0,.4); display: flex; align-items: center; justify-content: center; z-index: 200; padding: 24px; }
.wb-dlg { background: var(--bg-secondary); border-radius: var(--radius-lg); padding: 20px; width: 100%; max-width: 340px; }
.wb-dlg h3 { font-size: 16px; margin-bottom: 14px; color: var(--text-primary); }
.wb-dlg .input { width: 100%; margin-bottom: 16px; }
.wb-dlg-hint { font-size: 12px; color: var(--text-secondary); line-height: 1.6; margin: -6px 0 14px; text-align: left; }
.wb-dlg-btns { display: flex; gap: 10px; justify-content: flex-end; }
</style>
