<template>
  <div class="home" ref="homeRef">
    <!-- sticky 交叉淡化：每屏各占一个 wrapper、sticky 钉住（兼容老内核），后屏滑入时淡入盖住前屏 -->
    <div v-for="(s, i) in screens" :key="i" class="hs-wrap" :class="{ 'hs-wrap-last': i === screens.length - 1 }">
      <section class="hs" :style="{ zIndex: i + 1 }">
        <div class="hs-in">
          <h2 class="hs-line" v-html="s.line"></h2>
          <p class="hs-under" v-html="s.under"></p>
          <div v-if="s.stats" class="hs-stats">
            <div class="hs-stat"><b data-n="2041">2,041</b><span>今日字数</span></div>
            <div class="hs-stat"><b data-n="23">23</b><span>连续天数</span></div>
            <div class="hs-stat"><b data-n="412900">412,900</b><span>全书字数</span></div>
          </div>
          <div v-if="i === screens.length - 1" class="hs-cta">
            <!-- 小程序里一律给登录入口（网页残留的过期 token 会把按钮藏掉，导致用户卡死无出口） -->
            <button v-if="!isLoggedIn || inMiniProgram" class="btn primary big" @click="goLogin">开始写作</button>
            <router-link v-else to="/bookshelf" class="btn primary big">进入书架</router-link>
            <p class="hs-alt">
              <template v-if="!isLoggedIn && !inMiniProgram">已有账号？<router-link to="/login">登录</router-link> · </template>网页 / 手机 / 鸿蒙同步
            </p>
          </div>
        </div>
        <footer v-if="i === screens.length - 1" class="hs-icp">
          <a href="https://beian.mps.gov.cn/#/query/webSearch" target="_blank" rel="noopener">苏ICP备2026036211号-4</a>
        </footer>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { isMiniProgramEnv, mpNavigate } from '../core/mp'
const isLoggedIn = ref(!!localStorage.getItem('access_token'))
const inMiniProgram = isMiniProgramEnv
const router = useRouter()
const homeRef = ref<HTMLElement | null>(null)
// 小程序里登录走壳原生页（微信一键登录），浏览器里未登录直达注册
function goLogin() {
  if (inMiniProgram) mpNavigate('/pages/login/login')
  else router.push('/register')
}

const screens = [
  { line: '每个故事，<br>都从<em>一句</em>开始。<span class="hs-caret"></span>',
    under: '「雨下到第三天，巷口的灯笼终于灭了。」——一位作者的第一章第一句。<br>别管写得好不好，先写下来。' },
  { line: '卡住的时候，<br>让人物<em>自己走两步</em>。',
    under: '翻翻人物卡，看看上一章的结尾——故事比你更知道往哪走。' },
  { line: '<em>先写完</em>，<br>再写好。',
    under: '别删，别回头，完稿是一切修改的前提。<br>码字锁帮你管住那只总想删字的手。' },
  { line: '写下去，<br>是唯一的<em>捷径</em>。',
    under: '每日目标、连续打卡；一天两千字，一年就是一本书。', stats: true },
  { line: '深夜两点，<br>不止你<em>一盏灯</em>亮着。',
    under: '搭子在，拼字房的对手也在。<br>写故事的人，从来不是一个人。' },
  { line: '然后有一天，<br>它变成了<em>一本书</em>。',
    under: '卷章、人物、伏笔，一整本书的秩序都在；写完，导出带走。' },
]

// sticky 交叉淡化驱动：钉住的当前屏恒不透明；后屏从底部滑入时同步淡入（带底色盖住前屏），
// 盖满视口的瞬间恰好完全显形——任何时刻都是一幅完整画面
let cleanup: (() => void) | null = null
onMounted(() => {
  const home = homeRef.value
  if (!home) return
  // 优雅降级：老内核/异常浏览器退回静态整屏堆叠（.no-anim），保证任何环境都有完整内容
  const supportsSticky = typeof CSS !== 'undefined' && CSS.supports
    && (CSS.supports('position', 'sticky') || CSS.supports('position', '-webkit-sticky'))
  if (!supportsSticky || typeof requestAnimationFrame !== 'function') {
    home.classList.add('no-anim')
    return
  }
  try {
    mountDriver(home)
  } catch {
    home.classList.add('no-anim')
  }
})

function mountDriver(home: HTMLElement) {
  const wraps = [...home.querySelectorAll<HTMLElement>('.hs-wrap')]
  const clamp01 = (t: number) => Math.min(1, Math.max(0, t))
  const eased = (t: number) => 1 - Math.pow(1 - t, 3)
  let counted = false
  let ticking = false
  const update = () => {
    ticking = false
    const H = home.clientHeight
    const homeTop = home.getBoundingClientRect().top
    wraps.forEach((w) => {
      const inn = w.querySelector<HTMLElement>('.hs-in')
      const hs = w.querySelector<HTMLElement>('.hs')
      if (!inn || !hs) return
      const top = hs.getBoundingClientRect().top - homeTop // 滑入中 H→0；钉住 = 0；解钉后 <0
      const rel = w.getBoundingClientRect().top - homeTop
      const range = w.offsetHeight - H
      const pinP = range > 0 ? clamp01(-rel / range) : 0 // 钉住期进度 0→1
      let op: number, ty: number, sc: number
      if (top > 0) {
        // ① 上浮浮现：随屏从底部升起，进入视野后再淡入，钉住位（top=0）前完全显形
        const e = 1 - top / H
        const k = eased(clamp01((e - 0.1) / 0.55))
        op = k
        ty = (1 - k) * 84
        sc = 0.96 + 0.04 * k
      } else if (top > -2) {
        // ② 钉住停留：文字缓慢放大（呼吸感，不留死时间）
        op = 1
        ty = 0
        sc = 1 + 0.16 * pinP
      } else {
        // ③ 向前消失：只放大+淡出（朝镜头穿过），不带向上位移
        const x = clamp01(-top / (H * 0.8))
        op = 1 - eased(x)
        ty = 0
        sc = 1.16 * (1 + 0.35 * x)
      }
      inn.style.opacity = op.toFixed(3)
      inn.style.transform = `translateY(${ty.toFixed(1)}px) scale(${sc.toFixed(4)})`
    })
    // 数字墙：该屏钉住后开始滚动递增（一次性）
    if (!counted) {
      const w3 = wraps[3]
      const hs3 = w3?.querySelector<HTMLElement>('.hs')
      if (hs3 && hs3.getBoundingClientRect().top - homeTop < 4) {
        counted = true
        w3.querySelectorAll<HTMLElement>('.hs-stat b').forEach((b) => {
          const target = Number(b.dataset.n || '0')
          const t0 = performance.now()
          const dur = 900
          const step = (now: number) => {
            const k = Math.min(1, (now - t0) / dur)
            b.textContent = Math.round(target * (1 - Math.pow(1 - k, 3))).toLocaleString()
            if (k < 1) requestAnimationFrame(step)
          }
          requestAnimationFrame(step)
        })
      }
    }
  }
  const onScroll = () => {
    if (!ticking) { ticking = true; requestAnimationFrame(update) }
  }
  // 华为等老内核在内部滚动容器上可能不触发 scroll 事件——改用 rAF 常驻帧循环直接量位置驱动，
  // 不依赖任何事件；每帧仅 6 个 getBoundingClientRect+style 写入，且值不变时跳过写入
  const prev: string[] = []
  const loop = () => {
    let dirty = false
    wraps.forEach((w, i) => {
      const hs = w.querySelector<HTMLElement>('.hs')
      if (!hs) return
      const top = hs.getBoundingClientRect().top
      const key = i + ':' + Math.round(top)
      if (prev[i] !== key) { prev[i] = key; dirty = true }
    })
    if (dirty) update()
    cleanupRaf = requestAnimationFrame(loop)
  }
  let cleanupRaf = requestAnimationFrame(loop)
  home.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll)
  update()
  cleanup = () => {
    home.removeEventListener('scroll', onScroll)
    window.removeEventListener('resize', onScroll)
    cancelAnimationFrame(cleanupRaf)
  }
}
onBeforeUnmount(() => cleanup?.())
</script>

<style scoped>
/* sticky 交叉淡化（不用 fixed——华为等老内核浏览器在滚动容器内 fixed 不可靠） */
.home {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch; /* 老 iOS 内部容器惯性滚动 */
  width: 100%;
  position: relative;
}

/* 降级形态：不支持 sticky/rAF 的老浏览器 → 静态整屏堆叠，内容完整可读 */
.home.no-anim .hs-wrap,
.home.no-anim .hs-wrap-last { height: auto; }
.home.no-anim .hs { position: relative; height: auto; min-height: calc(100vh - 56px); }
.home.no-anim .hs-in { opacity: 1 !important; transform: none !important; }

.hs-wrap {
  height: 132vh;
  position: relative;
}

.hs-wrap-last {
  height: 120vh;
}

.hs {
  position: -webkit-sticky;
  position: sticky;
  top: 0;
  /* 100vh 兜底在前、100dvh 在后：老内核吃到 vh，新内核吃到 dvh */
  height: calc(100vh - var(--nav-height, 56px));
  height: calc(100dvh - var(--nav-height, 56px));
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 0 32px;
  overflow: hidden;
  background: var(--bg-primary); /* 淡入的后屏必须带底色，才能盖住前屏文字 */
}

.hs-in {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  will-change: transform, opacity;
}

/* JS 未跑/失败时首屏仍可见 */
.hs-wrap:not(:first-child) .hs-in { opacity: 0; }

.hs-line {
  font-family: var(--font-serif);
  font-weight: 700;
  /* clamp 老浏览器（Safari<13.1 等）不认：先给固定值兜底再上 clamp */
  font-size: 42px;
  font-size: clamp(30px, 5.4vw, 60px);
  line-height: 1.6;
  letter-spacing: 2px;
  color: var(--text-primary);
  margin-bottom: 28px;
}

.hs-line :deep(em) {
  font-style: normal;
  color: #6aa877;
}

.hs-caret {
  display: inline-block;
  width: 3px;
  height: 0.82em;
  background: #6aa877;
  animation: hs-blink 1.1s step-end infinite;
  vertical-align: -0.06em;
  margin-left: 6px;
}

@keyframes hs-blink {
  50% { opacity: 0; }
}

.hs-under {
  /* clamp 兜底：老浏览器固定值在前 */
  font-size: 15px;
  font-size: clamp(13px, 1.5vw, 16px);
  color: var(--text-secondary);
  line-height: 2.1;
  max-width: 560px;
}

.hs-stats {
  display: flex;
  gap: clamp(36px, 6vw, 76px);
  margin-top: 42px;
  flex-wrap: wrap;
  justify-content: center;
}

.hs-stat b {
  display: block;
  font-family: var(--font-serif);
  font-size: 38px;
  font-size: clamp(30px, 4vw, 46px);
  font-weight: 700;
  color: #6aa877;
  line-height: 1.2;
}

.hs-stat span {
  font-size: 13px;
  color: var(--text-secondary);
  letter-spacing: 2px;
}

.hs-cta {
  margin-top: 42px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.hs-cta .btn.big {
  width: auto;
  padding: 13px 44px;
  font-size: 16px;
}

.hs-alt {
  margin-top: 14px;
  font-size: 13px;
  color: var(--text-secondary);
}

.hs-alt a {
  color: #6aa877;
  text-decoration: none;
}

.hs-icp {
  position: absolute;
  bottom: 14px;
  width: 100%;
  text-align: center;
  font-size: 12px;
}

.hs-icp a {
  color: var(--text-secondary);
  text-decoration: none;
  opacity: 0.6;
}

.hs-icp a:hover { opacity: 1; }

/* 手机/窄屏：长句不强制断行 */
@media (max-width: 720px) {
  .hs { padding: 0 22px; }
  .hs-line { line-height: 1.7; }
  .hs-under { line-height: 2; }
  .hs-under :deep(br) { display: none; }
}

/* 减弱动态：只关光标闪烁（滚动淡入是联动效果不关，见 script 注释） */
@media (prefers-reduced-motion: reduce) {
  .hs-caret { animation: none; }
}
</style>
