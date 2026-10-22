<template>
  <Teleport to="body">
    <!-- ====== Welcome prompt ====== -->
    <div v-if="showPrompt" class="tour-overlay" @click.self="dismissPrompt">
      <div class="welcome-card" @click.stop>
        <div class="welcome-icon">🎓</div>
        <div class="welcome-title">欢迎来到写作菌</div>
        <div class="welcome-desc">花 1 分钟了解编辑器的核心功能，快速上手 AI 辅助写作。</div>
        <div class="welcome-actions">
          <button class="welcome-later" @click="dismissPrompt">以后再说</button>
          <button class="welcome-go" @click="startGuide">开始引导</button>
        </div>
      </div>
    </div>

    <!-- ====== Spotlight overlay (for real DOM steps) ====== -->
    <div v-if="active && !demoMode" class="tour-overlay" @click.self="finish">
      <div class="tour-spotlight" :style="spotlightStyle"></div>
      <div class="tour-tooltip" :class="tourArrow" :style="tooltipStyle" @click.stop>
        <div class="tour-badge">{{ currentStepData.badge }}</div>
        <div class="tour-title">{{ currentStepData.title }}</div>
        <div class="tour-desc">{{ currentStepData.desc }}</div>
        <div v-if="currentStepData.features?.length" class="tour-features">
          <div v-for="f in currentStepData.features" :key="f.name" class="tour-feature">
            <div class="tf-icon" :style="{ background: f.bg }">{{ f.icon }}</div>
            <div class="tf-body"><div class="tf-name">{{ f.name }}</div><div class="tf-desc">{{ f.desc }}</div></div>
          </div>
        </div>
        <div class="tour-footer">
          <div class="tour-dots">
            <span v-for="(_, i) in allSteps" :key="i" class="tour-dot" :class="{ active: i === currentStep }"></span>
          </div>
          <div style="display:flex;gap:8px;">
            <button class="tour-skip" @click="finish">跳过</button>
            <button class="tour-next" @click="nextStep">{{ currentStepData.isLast ? '开始演示 →' : '下一步' }}</button>
          </div>
        </div>
      </div>
    </div>

    <!-- ====== AI Demo overlay (mock editor) ====== -->
    <div v-if="active && demoMode" class="demo-stage">
      <div v-show="demoSpotVisible" class="demo-mask">
        <div class="demo-spotlight" :style="demoSpotStyle"></div>
      </div>
      <div v-if="demoCaption" class="demo-caption show">
        <span class="cap-dot"></span><span>{{ demoCaption }}</span>
      </div>
      <!-- Mock editor layout -->
      <div class="demo-layout">
        <div class="demo-sidebar">
          <div class="demo-branch"><span>故事线</span><span class="demo-branch-name">主线</span></div>
          <div class="demo-tree">
            <div class="demo-zone-label warm">设定</div>
            <div class="demo-tree-item"><span class="dt-icon">📋</span> 大纲</div>
            <div class="demo-tree-item"><span class="dt-icon">👤</span> 人物</div>
            <div class="demo-tree-item"><span class="dt-icon">🌍</span> 世界观</div>
            <div class="demo-zone-divider"></div>
            <div class="demo-zone-label">文章</div>
            <div class="demo-tree-item active"><span class="dt-icon">📖</span> 第一章 · 雨夜</div>
            <div class="demo-tree-item"><span class="dt-icon">📖</span> 第二章 · 迷踪</div>
            <div class="demo-tree-item"><span class="dt-icon">📖</span> 第三章 · 真相</div>
          </div>
        </div>
        <div class="demo-main">
          <div class="demo-header">
            <span class="demo-back">← 返回书架</span>
            <span class="demo-proj-name">雨夜追踪</span>
            <span class="demo-doc-sep">·</span>
            <span class="demo-chapter">第一章 · 雨夜</span>
            <div class="demo-modes">
              <span class="demo-mode">构思</span>
              <span class="demo-mode active">写作</span>
            </div>
            <div class="demo-header-actions">
              <span class="demo-icon-btn">🌙</span><span class="demo-icon-btn">📖</span><span class="demo-icon-btn">⬜</span><span class="demo-icon-btn">⏱</span>
            </div>
          </div>
          <div class="demo-editor">
            <p v-html="para1HTML"></p>
            <p v-html="para2HTML"></p>
            <p v-html="demoOutput"></p>
          </div>
        </div>
      </div>
      <div v-show="aiToolbarShow" class="demo-ai-toolbar" :style="aiToolbarStyle">
        <span v-for="a in ['continue','rewrite','polish','expand']" :key="a" class="ai-btn"
          :class="{ highlight: aiHighlightBtn === a }" @click="demoActionClick(a)">{{ aiBtnLabel(a) }}</span>
      </div>
      <div v-show="aiBottombarShow" class="demo-ai-bottombar" v-html="aiBottombarHTML"></div>
      <div v-if="demoPulseStyle" class="pulse-ring" :style="demoPulseStyle"></div>
      <div v-if="demoPointerStyle" class="click-pointer" :style="demoPointerStyle">👆</div>
      <div v-if="fakeCursorShow" class="fake-cursor" :style="fakeCursorStyle">🖱️</div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const STORAGE_KEY = 'xiezuojun_onboarded'

const active = ref(false)
const currentStep = ref(0)
const demoMode = ref(false)
const showPrompt = ref(false)

// ====== Spotlight on real DOM ======
const spotlightStyle = ref({})
const tooltipStyle = ref({})
const tourArrow = ref('')
const spotRect = ref({ x: 0, y: 0, w: 0, h: 0 })

interface Feature { icon: string; bg: string; name: string; desc: string }
interface Step {
  badge: string; title: string; desc: string;
  selector?: string; placement?: string;
  features?: Feature[]; isDemo?: boolean; isLast?: boolean;
}

// Steps 1-4: highlight real DOM. Step 5: AI demo overlay.
const allSteps: Step[] = [
  { badge: '第 1 步 / 共 5 步', title: '左侧栏：设定 + 文章', desc: '左侧分两个区域，所有创作素材和正文章节都在这里管理。',
    selector: '.doc-tree', placement: 'right',
    features: [{ icon:'📋', bg:'#f7f4f1', name:'设定区', desc:'放创作素材：大纲、人物设定、世界观' },{ icon:'📖', bg:'#fff', name:'文章区', desc:'按章节组织的正文，点击即写' }] },
  { badge: '第 2 步 / 共 5 步', title: '顶部：两种工作模式', desc: '随时切换模式，从构思到成稿。',
    selector: '.mode-switcher', placement: 'bottom',
    features: [{ icon:'🧠', bg:'#af52de15', name:'构思', desc:'5步AI引导：梗概→世界观→人物→大纲' },{ icon:'✍️', bg:'#007aff15', name:'写作', desc:'沉浸式编辑器，AI续写/改写/润色/扩写' }] },
  { badge: '第 3 步 / 共 5 步', title: '右侧工具：分屏 · 专注 · 版本历史', desc: '写作模式下，右侧工具栏提供多种辅助功能。',
    selector: '.header-actions', placement: 'bottom',
    features: [{ icon:'📖', bg:'#007aff15', name:'分屏参考', desc:'左边写正文，右边看设定素材' },{ icon:'⬜', bg:'#007aff15', name:'专注模式', desc:'隐藏侧栏，全屏写作' },{ icon:'⏱', bg:'#007aff15', name:'版本历史', desc:'随时回到任意历史版本，不怕写错' }] },
  { badge: '第 4 步 / 共 5 步', title: '编辑区：选中文字唤起 AI', desc: '中间是正文编辑区。选中任意文字，上方会弹出 AI 工具栏。接下来演示完整流程：',
    selector: '.editor-wrapper', placement: 'right', isLast: true },
  { badge: '第 5 步 / 共 5 步', title: 'AI 写作辅助演示', desc: '选中文字 → 选操作 → 输入方向 → AI 生成 → 采用/丢弃', isDemo: true },
]

const currentStepData = computed(() => allSteps[currentStep.value] || {})

// ====== Spotlight helpers ======
function positionSpotlight(selector: string) {
  const el = document.querySelector(selector) as HTMLElement
  if (!el) return false
  const r = el.getBoundingClientRect()
  const pad = 6
  const x = r.left - pad, y = r.top - pad, w = r.width + pad * 2, h = r.height + pad * 2
  spotRect.value = { x, y, w, h }
  spotlightStyle.value = { left: x + 'px', top: y + 'px', width: w + 'px', height: h + 'px' }
  return true
}

function positionTooltip(placement: string) {
  const tw = 380
  const tip = document.querySelector('.tour-tooltip') as HTMLElement
  const th = tip?.offsetHeight || 250
  const s = spotRect.value
  const r = { left: s.x + 6, top: s.y + 6, width: s.w - 12, height: s.h - 12, right: s.x + s.w - 6, bottom: s.y + s.h - 6 }
  let tx = 0, ty = 0, arrow = ''
  switch (placement) {
    case 'bottom': tx = r.left + r.width / 2 - tw / 2; ty = r.bottom + 20; arrow = 'arrow-top'; break
    case 'right': tx = r.right + 20; ty = r.top + r.height / 2 - th / 2; arrow = 'arrow-left'; break
    case 'left': tx = r.left - tw - 20; ty = r.top + r.height / 2 - th / 2; arrow = 'arrow-right'; break
    case 'top': tx = r.left + r.width / 2 - tw / 2; ty = r.top - th - 20; arrow = 'arrow-bottom'; break
  }
  tx = Math.max(16, Math.min(tx, window.innerWidth - tw - 16))
  ty = Math.max(16, Math.min(ty, window.innerHeight - th - 16))
  tooltipStyle.value = { left: tx + 'px', top: ty + 'px' }
  tourArrow.value = arrow
}

function showSpotlightStep() {
  const s = allSteps[currentStep.value]
  if (!s || !s.selector) return
  nextTick(() => {
    let retries = 5
    const tryShow = () => {
      const el = document.querySelector(s.selector!) as HTMLElement
      if (el) {
        if (positionSpotlight(s.selector!)) {
          nextTick(() => positionTooltip(s.placement || 'bottom'))
        }
        return
      }
      if (retries-- > 0) { setTimeout(tryShow, 200); return }
      // fallback: center
      spotRect.value = { x: 0, y: 0, w: 0, h: 0 }
      spotlightStyle.value = { left: '0px', top: '0px', width: '0px', height: '0px' }
      tooltipStyle.value = { left: Math.max(16, (window.innerWidth - 380) / 2) + 'px', top: Math.max(16, (window.innerHeight - 250) / 2) + 'px' }
      tourArrow.value = ''
    }
    tryShow()
  })
}

function nextStep() {
  if (currentStep.value < allSteps.length - 1) {
    currentStep.value++
    const s = allSteps[currentStep.value]
    if (s.isDemo) {
      demoMode.value = true
      startTypingPhase()
    } else {
      demoMode.value = false
      showSpotlightStep()
    }
  } else {
    finish()
  }
}

// ====== AI Demo (overlay mock) ======
const demoCaption = ref('')
const para1HTML = ref('')
const para2HTML = ref('')
const demoOutput = ref('')
const demoSpotVisible = ref(false)
const demoSpotStyle = ref({})
const demoPulseStyle = ref<Record<string, string> | null>(null)
const demoPointerStyle = ref<Record<string, string> | null>(null)
const fakeCursorShow = ref(false)
const fakeCursorStyle = ref({})
const aiToolbarShow = ref(false)
const aiToolbarStyle = ref({})
const aiHighlightBtn = ref('')
const aiBottombarShow = ref(false)
const aiBottombarHTML = ref('')
let timers: number[] = []

const para1Text = '夜色笼罩了整座城市，霓虹灯在雨幕中模糊成一片光晕，她独自站在天桥上，望着下方川流不息的车河。'
const para2Text = '手机震动了一下。是一条来自未知号码的消息："别回头。"'
const selectText = '霓虹灯在雨幕中模糊成一片光晕'
const aiResult = '雨滴穿过霓虹的光谱，将整条街渲染成一幅流动的水彩——橙红、靛蓝、暖黄，在湿润的柏油路面上交融、碎裂、又重新凝聚。'

function aiBtnLabel(a: string) { return { continue: '续写', rewrite: '改写', polish: '润色', expand: '扩写' }[a] || a }
function clearTimers() { timers.forEach(t => { clearInterval(t); clearTimeout(t) }); timers = [] }
function getStageRect() { return (document.querySelector('.demo-stage') as HTMLElement)?.getBoundingClientRect() || { left: 0, top: 0 } }

function demoSpotOn(el: HTMLElement | null) {
  if (!el) { demoSpotVisible.value = false; return }
  const r = el.getBoundingClientRect(), s = getStageRect(), pad = 12
  demoSpotStyle.value = { left: (r.left - s.left - pad) + 'px', top: (r.top - s.top - pad) + 'px', width: (r.width + pad*2) + 'px', height: (r.height + pad*2) + 'px' }
  demoSpotVisible.value = true
}
function demoSpotOff() { demoSpotVisible.value = false }
function guide(el: HTMLElement | null) {
  if (!el) return
  const r = el.getBoundingClientRect(), s = getStageRect()
  demoPulseStyle.value = { left: (r.left - s.left - 4) + 'px', top: (r.top - s.top - 4) + 'px', width: (r.width + 8) + 'px', height: (r.height + 8) + 'px' }
  demoPointerStyle.value = { left: (r.left - s.left + r.width/2 - 8) + 'px', top: (r.bottom - s.top + 4) + 'px' }
}
function unguide() { demoPulseStyle.value = null; demoPointerStyle.value = null }
function moveFakeCursor(x: number, y: number) { fakeCursorShow.value = true; fakeCursorStyle.value = { left: x+'px', top: y+'px' } }
function hideFakeCursor() { fakeCursorShow.value = false }
function waitForClick(el: HTMLElement, fn: () => void) {
  el.style.cursor = 'pointer'
  const h = () => { el.removeEventListener('click', h); el.style.cursor = ''; fn() }
  el.addEventListener('click', h)
}

function escapeChar(c: string) { return c === '"' ? '&quot;' : c === '<' ? '&lt;' : c === '>' ? '&gt;' : c }

function formatTypingProgress(typed: string) {
  const selStart = para1Text.indexOf(selectText), selEnd = selStart + selectText.length
  let html = ''
  for (let i = 0; i < typed.length; i++) {
    if (i === selStart) html += '<span class="sel-highlight" id="selText">'
    html += escapeChar(typed[i])
    if (i === selEnd - 1) html += '</span>'
  }
  return html + '<span class="typing-cursor-inline"></span>'
}

function formatPara(text: string) {
  const selStart = text.indexOf(selectText), selEnd = selStart + selectText.length
  let html = ''
  for (let i = 0; i < text.length; i++) {
    if (i === selStart) html += '<span class="sel-highlight" id="selText">'
    html += escapeChar(text[i])
    if (i === selEnd - 1) html += '</span>'
  }
  return html
}

function startTypingPhase() {
  demoCaption.value = '正在创作第一章 · 雨夜…'
  para1HTML.value = '<span class="typing-cursor-inline"></span>'
  para2HTML.value = ''
  demoOutput.value = ''
  aiToolbarShow.value = false
  aiBottombarShow.value = false
  demoSpotVisible.value = false
  unguide(); hideFakeCursor()

  let i = 0
  const typeT = setInterval(() => {
    i++
    if (i >= para1Text.length) {
      clearInterval(typeT)
      para1HTML.value = formatPara(para1Text)
      const t1 = setTimeout(() => {
        para2HTML.value = '<span class="typing-cursor-inline"></span>'
        let j = 0
        const typeT2 = setInterval(() => {
          j++
          if (j >= para2Text.length) {
            clearInterval(typeT2)
            para2HTML.value = para2Text
            const t2 = setTimeout(() => startSelectPhase(), 800)
            timers.push(t2 as unknown as number)
          }
          para2HTML.value = para2Text.slice(0, j) + '<span class="typing-cursor-inline"></span>'
        }, 50)
        timers.push(typeT2 as unknown as number)
      }, 400)
      timers.push(t1 as unknown as number)
      return
    }
    para1HTML.value = formatTypingProgress(para1Text.slice(0, i))
  }, 60)
  timers.push(typeT as unknown as number)
}

function startSelectPhase() {
  demoCaption.value = '选中你想修改的文字，AI 就会弹出来'
  nextTick(() => {
    const selEl = document.getElementById('selText')
    if (!selEl) return
    const r = selEl.getBoundingClientRect(), s = getStageRect()
    moveFakeCursor(r.left - s.left, r.top - s.top + r.height/2 - 10)
    const t = setTimeout(() => {
      demoCaption.value = '按住鼠标，拖选文字…'
      selEl.style.transition = 'background .8s, box-shadow .8s'
      selEl.classList.add('active')
      const startX = r.left - s.left, endX = r.right - s.left, steps = 20
      let step = 0
      const dragT = setInterval(() => {
        step++
        moveFakeCursor(startX + (endX - startX) * (step/steps), r.top - s.top + r.height/2 - 10)
        if (step >= steps) {
          clearInterval(dragT)
          demoCaption.value = '✓ 选中文字后，上方弹出 AI 工具栏'
          hideFakeCursor()
          const t2 = setTimeout(() => startAIToolbarPhase(), 600)
          timers.push(t2 as unknown as number)
        }
      }, 30)
      timers.push(dragT as unknown as number)
    }, 800)
    timers.push(t as unknown as number)
  })
}

function startAIToolbarPhase() {
  const selEl = document.getElementById('selText')
  if (!selEl) return
  const r = selEl.getBoundingClientRect()
  aiToolbarStyle.value = { left: (r.left + r.width/2 - 100) + 'px', top: (r.top - 50) + 'px' }
  aiToolbarShow.value = true
  const t = setTimeout(() => {
    demoCaption.value = '点击「改写」（续写/润色/扩写操作一样）'
    const tb = document.querySelector('.demo-ai-toolbar') as HTMLElement
    demoSpotOn(tb)
    aiHighlightBtn.value = 'rewrite'
    const rewriteBtn = tb.querySelector('.ai-btn.highlight') as HTMLElement
    const t2 = setTimeout(() => guide(rewriteBtn), 300)
    timers.push(t2 as unknown as number)
  }, 800)
  timers.push(t as unknown as number)
}

function demoActionClick(action: string) {
  if (action !== 'rewrite' || aiHighlightBtn.value !== 'rewrite') return
  unguide(); demoSpotOff()
  aiToolbarShow.value = false
  aiHighlightBtn.value = ''
  const selEl = document.getElementById('selText')
  if (selEl) { selEl.classList.remove('active'); selEl.classList.add('strike') }
  demoCaption.value = '✓ 原文加了删除线，底部出现方向输入框'
  aiBottombarShow.value = true
  aiBottombarHTML.value = '<button class="bar-btn">✕</button><input class="bar-input" placeholder="改写方向（可选），按回车生成" /><button class="bar-btn primary" id="demoGen">▶ 生成</button>'

  const t = setTimeout(() => {
    demoCaption.value = '输入改写方向后点「生成」（不输也行，直接回车）'
    const bb = document.querySelector('.demo-ai-bottombar') as HTMLElement
    demoSpotOn(bb)
    const genBtn = document.getElementById('demoGen')!
    const input = document.querySelector('.demo-ai-bottombar .bar-input') as HTMLInputElement
    const t2 = setTimeout(() => guide(genBtn), 300)
    timers.push(t2 as unknown as number)
    const text = '更加细腻一点'
    let i = 0
    const typeT = setInterval(() => { if (i >= text.length) { clearInterval(typeT); return } input.value = text.slice(0, ++i) }, 120)
    timers.push(typeT as unknown as number)

    function doGenerate() {
      unguide(); demoSpotOff()
      demoCaption.value = '④ AI 正在生成…'
      aiBottombarHTML.value = '<span class="bar-hint">AI 生成中…</span><button class="bar-btn danger">⏹ 停止</button>'
      let j = 0
      const genT = setInterval(() => {
        if (j >= aiResult.length) {
          clearInterval(genT)
          demoCaption.value = '⑤ 点击「✓ 采用」替换原文'
          demoOutput.value = '<span class="demo-text-ai">' + aiResult + '</span>'
          aiBottombarHTML.value = '<span class="bar-hint">已生成</span><button class="bar-btn">↻ 重试</button><button class="bar-btn danger" id="demoDiscard">🗑 丢弃</button><button class="bar-btn success" id="demoAccept">✓ 采用</button>'
          const t3 = setTimeout(() => {
            const bb2 = document.querySelector('.demo-ai-bottombar') as HTMLElement
            demoSpotOn(bb2)
            const acceptBtn = document.getElementById('demoAccept')!
            const discardBtn = document.getElementById('demoDiscard')
            guide(acceptBtn)
            if (discardBtn) {
              waitForClick(discardBtn, () => {
                unguide(); demoSpotOff()
                demoCaption.value = '✅ 已丢弃，原文保留。演示完毕！'
                const sel = document.getElementById('selText')
                if (sel) sel.classList.remove('strike')
                demoOutput.value = ''
                aiBottombarShow.value = false
                finishDemo()
              })
            }
            waitForClick(acceptBtn, () => {
              unguide(); demoSpotOff()
              demoCaption.value = '✅ 已采用！AI 文本替换了原文。演示完毕！'
              const sel = document.getElementById('selText')
              if (sel) {
                sel.classList.add('fade-out')
                const t4 = setTimeout(() => {
                  sel.classList.remove('strike','fade-out','sel-highlight','active')
                  sel.classList.add('demo-text-ai')
                  sel.textContent = aiResult
                  demoOutput.value = ''
                  aiBottombarShow.value = false
                  finishDemo()
                }, 500)
                timers.push(t4 as unknown as number)
              } else { finishDemo() }
            })
          }, 500)
          timers.push(t3 as unknown as number)
        }
        j++
        demoOutput.value = '<span class="demo-text-ai">' + aiResult.slice(0, j) + '<span class="typing-cursor"></span></span>'
      }, 40)
      timers.push(genT as unknown as number)
    }
    genBtn.addEventListener('click', doGenerate, { once: true })
    input.addEventListener('keydown', (e) => { if (e.key === 'Enter') doGenerate() })
  }, 600)
  timers.push(t as unknown as number)
}

function finishDemo() {
  const t1 = setTimeout(() => { demoCaption.value = '' }, 3000)
  const t2 = setTimeout(() => { finish() }, 4000)
  timers.push(t1 as unknown as number, t2 as unknown as number)
}

// ====== Start / finish ======
function showWelcomePrompt() {
  if (localStorage.getItem(STORAGE_KEY) === '1') return
  showPrompt.value = true
}

function startGuide() {
  showPrompt.value = false
  start(true)
}

function dismissPrompt() {
  showPrompt.value = false
  localStorage.setItem(STORAGE_KEY, '1')
}

function start(force = false) {
  if (!force && localStorage.getItem(STORAGE_KEY) === '1') return
  if (active.value) return
  active.value = true
  demoMode.value = false
  currentStep.value = 0
  clearTimers()
  showSpotlightStep()
}

function finish() {
  clearTimers()
  active.value = false
  demoMode.value = false
  demoCaption.value = ''
  localStorage.setItem(STORAGE_KEY, '1')
}

watch(() => route.name, (name) => {
  if (!localStorage.getItem('access_token')) return
  if (localStorage.getItem(STORAGE_KEY) === '1') return
  if (!active.value && !showPrompt.value && name === 'Editor') {
    setTimeout(() => showWelcomePrompt(), 800)
  }
})

defineExpose({ start, forceStart: () => { showPrompt.value = false; start(true) } })

onMounted(() => {
  if (!localStorage.getItem('access_token')) return
  if (localStorage.getItem(STORAGE_KEY) === '1') return
  if (route.name !== 'Editor') return
  setTimeout(() => showWelcomePrompt(), 800)
})

onUnmounted(() => { clearTimers() })
</script>

<style scoped>
/* ====== Spotlight overlay ====== */
.tour-overlay { position: fixed; inset: 0; z-index: 9999; pointer-events: none; }

/* Welcome card */
.welcome-card {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  background: var(--bg-secondary, #fff); border-radius: 20px;
  padding: 40px 36px 28px; width: 380px; max-width: 92vw;
  box-shadow: 0 20px 60px rgba(0,0,0,.2); pointer-events: auto;
  text-align: center; color: var(--text-primary, #1d1d1f);
  animation: welcomeIn .35s cubic-bezier(.25,.46,.45,.94);
}
@keyframes welcomeIn { from { opacity: 0; transform: translate(-50%, -45%) scale(.95); } to { opacity: 1; transform: translate(-50%, -50%) scale(1); } }
.welcome-icon { font-size: 48px; margin-bottom: 16px; }
.welcome-title { font-family: var(--font-serif, serif); font-size: 22px; font-weight: 600; margin-bottom: 8px; }
.welcome-desc { font-size: 14px; line-height: 1.6; color: var(--text-secondary, #86868b); margin-bottom: 24px; }
.welcome-actions { display: flex; gap: 12px; justify-content: center; }
.welcome-later { padding: 10px 24px; border: 1px solid var(--border, #e5e5ea); background: transparent; color: var(--text-secondary, #86868b); font-size: 14px; border-radius: 10px; cursor: pointer; }
.welcome-go { padding: 10px 28px; border: none; background: var(--accent, #007aff); color: #fff; font-size: 14px; font-weight: 600; border-radius: 10px; cursor: pointer; }
.tour-spotlight { position: absolute; border-radius: 12px; box-shadow: 0 0 0 9999px rgba(0,0,0,0.55); transition: all .35s cubic-bezier(.25,.46,.45,.94); z-index: 1; pointer-events: none; }
.tour-tooltip {
  position: absolute; background: var(--bg-secondary, #fff); border-radius: 14px;
  padding: 20px 24px 16px; width: 380px; max-width: 92vw;
  box-shadow: 0 12px 40px rgba(0,0,0,.25); z-index: 2;
  transition: all .35s cubic-bezier(.25,.46,.45,.94);
  pointer-events: auto; color: var(--text-primary, #1d1d1f);
}
.tour-tooltip::before { content: ''; position: absolute; width: 14px; height: 14px; background: var(--bg-secondary, #fff); transform: rotate(45deg); }
.tour-tooltip.arrow-top::before { top: -7px; left: 50%; margin-left: -7px; }
.tour-tooltip.arrow-bottom::before { bottom: -7px; left: 50%; margin-left: -7px; }
.tour-tooltip.arrow-left::before { left: -7px; top: 50%; margin-top: -7px; }
.tour-tooltip.arrow-right::before { right: -7px; top: 50%; margin-top: -7px; }
.tour-badge { font-size: 11px; font-weight: 600; color: var(--accent, #007aff); letter-spacing: 0.5px; margin-bottom: 6px; }
.tour-title { font-family: var(--font-serif, serif); font-size: 17px; font-weight: 600; margin-bottom: 8px; }
.tour-desc { font-size: 13px; line-height: 1.7; color: var(--text-secondary, #86868b); margin-bottom: 12px; }
.tour-features { margin-bottom: 14px; }
.tour-feature { display: flex; gap: 10px; padding: 6px 0; border-bottom: 1px solid rgba(0,0,0,0.05); }
.tour-feature:last-child { border-bottom: none; }
.tf-icon { width: 28px; height: 28px; border-radius: 7px; display: flex; align-items: center; justify-content: center; font-size: 14px; flex-shrink: 0; }
.tf-body { flex: 1; }
.tf-name { font-size: 13px; font-weight: 600; }
.tf-desc { font-size: 12px; color: var(--text-secondary, #86868b); line-height: 1.5; }
.tour-footer { display: flex; align-items: center; justify-content: space-between; }
.tour-dots { display: flex; gap: 5px; }
.tour-dot { width: 6px; height: 6px; border-radius: 50%; background: #d1d1d6; transition: all .2s; }
.tour-dot.active { width: 18px; border-radius: 3px; background: var(--accent, #007aff); }
.tour-skip { padding: 6px 12px; border: none; background: transparent; color: var(--text-secondary, #86868b); font-size: 13px; cursor: pointer; border-radius: 6px; }
.tour-next { padding: 6px 18px; border: none; background: var(--accent, #007aff); color: #fff; font-size: 13px; font-weight: 600; border-radius: 6px; cursor: pointer; }

/* ====== AI Demo overlay ====== */
.demo-stage { position: fixed; inset: 0; z-index: 9999; background: var(--bg-primary, #f5f5f7); display: flex; flex-direction: column; }
.demo-mask { position: absolute; inset: 0; z-index: 55; pointer-events: none; }
.demo-spotlight { position: absolute; border-radius: 10px; box-shadow: 0 0 0 9999px rgba(0,0,0,0.55); transition: all .4s cubic-bezier(.25,.46,.45,.94); }
.demo-caption {
  position: absolute; top: 16px; left: 50%; transform: translateX(-50%);
  background: var(--bg-secondary, #fff); border-radius: 10px; padding: 10px 20px;
  font-size: 14px; font-weight: 500; color: var(--accent, #007aff);
  box-shadow: 0 2px 12px rgba(0,0,0,.1); z-index: 70;
  display: flex; align-items: center; gap: 8px; white-space: nowrap; max-width: 90vw;
}
.cap-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent, #007aff); animation: capPulse 1.2s infinite; flex-shrink: 0; }
@keyframes capPulse { 0%,100%{opacity:1;} 50%{opacity:.3;} }

.demo-layout { display: flex; flex: 1; overflow: hidden; }
.demo-sidebar { width: 240px; flex-shrink: 0; background: var(--bg-secondary, #fff); border-right: 1px solid var(--border, #e5e5ea); display: flex; flex-direction: column; font-size: 13px; }
.demo-branch { padding: 10px 14px; border-bottom: 1px solid var(--border, #e5e5ea); display: flex; align-items: center; gap: 6px; color: var(--text-secondary, #86868b); }
.demo-branch-name { color: var(--text-primary, #1d1d1f); font-weight: 500; }
.demo-tree { padding: 8px 0; flex: 1; overflow: hidden; }
.demo-zone-label { padding: 4px 14px; font-size: 11px; font-weight: 600; color: var(--text-tertiary, #aeaeb2); text-transform: uppercase; letter-spacing: 1px; }
.demo-zone-label.warm { color: #b8860b; }
.demo-zone-divider { height: 1px; background: var(--border, #e5e5ea); margin: 6px 10px; }
.demo-tree-item { padding: 5px 14px; cursor: default; color: var(--text-secondary, #86868b); display: flex; align-items: center; gap: 6px; line-height: 1.4; }
.demo-tree-item.active { background: rgba(0,122,255,0.08); color: var(--accent, #007aff); font-weight: 500; border-left: 3px solid var(--accent, #007aff); padding-left: 11px; }
.dt-icon { font-size: 13px; flex-shrink: 0; }

.demo-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.demo-header { height: 48px; flex-shrink: 0; display: flex; align-items: center; gap: 8px; padding: 0 16px; border-bottom: 1px solid var(--border, #e5e5ea); background: var(--bg-secondary, #fff); font-size: 13px; }
.demo-back { color: var(--text-secondary, #86868b); }
.demo-proj-name { font-weight: 600; color: var(--text-primary, #1d1d1f); }
.demo-doc-sep { color: var(--text-tertiary, #aeaeb2); }
.demo-chapter { color: var(--text-secondary, #86868b); }
.demo-modes { display: flex; gap: 2px; margin-left: auto; background: var(--bg-primary, #f5f5f7); border-radius: 8px; padding: 2px; }
.demo-mode { padding: 4px 12px; border-radius: 6px; font-size: 12px; font-weight: 500; color: var(--text-secondary, #86868b); }
.demo-mode.active { background: var(--bg-secondary, #fff); color: var(--accent, #007aff); box-shadow: 0 1px 3px rgba(0,0,0,.08); }
.demo-header-actions { display: flex; gap: 4px; margin-left: 8px; }
.demo-icon-btn { width: 30px; height: 30px; border-radius: 7px; display: flex; align-items: center; justify-content: center; font-size: 14px; color: var(--text-secondary, #86868b); }

.demo-editor { padding: 50px 70px 40px; flex: 1; overflow-y: auto; font-family: var(--font-serif, serif); font-size: 17px; line-height: 2; color: var(--text-primary, #1d1d1f); position: relative; }
.demo-editor p { margin-bottom: 20px; position: relative; }
:deep(.typing-cursor-inline)::after { content: '▋'; animation: blink .6s infinite; color: var(--accent, #007aff); }
@keyframes blink { 50% { opacity: 0; } }
:deep(.sel-highlight) { border-radius: 2px; transition: background .15s, box-shadow .15s; }
:deep(.sel-highlight.active) { background: rgba(0,122,255,0.25); box-shadow: 0 0 0 2px rgba(0,122,255,0.3); }
:deep(.sel-highlight.strike) { text-decoration: line-through; color: var(--text-tertiary, #aeaeb2); }
:deep(.sel-highlight.fade-out) { opacity: 0; transition: opacity .3s; }
:deep(.demo-text-ai) { background: rgba(52,199,89,0.15); border-radius: 3px; padding: 1px 3px; }
:deep(.typing-cursor)::after { content: '▋'; animation: blink .6s infinite; color: var(--accent, #007aff); }

.fake-cursor { position: absolute; z-index: 65; pointer-events: none; font-size: 20px; transition: all .3s cubic-bezier(.25,.46,.45,.94); filter: drop-shadow(0 2px 4px rgba(0,0,0,.2)); }
.demo-ai-toolbar { position: absolute; display: flex; background: var(--bg-secondary, #fff); border-radius: 10px; box-shadow: 0 4px 16px rgba(0,0,0,.15); padding: 4px; z-index: 61; }
.demo-ai-toolbar .ai-btn { padding: 6px 14px; font-size: 13px; font-weight: 500; border-radius: 7px; cursor: pointer; position: relative; z-index: 61; }
.demo-ai-toolbar .ai-btn.highlight { background: var(--accent, #007aff); color: #fff; }
.demo-ai-bottombar { position: absolute; bottom: 36px; left: 50%; transform: translateX(-50%); display: flex; background: var(--bg-secondary, #fff); border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,.15); padding: 8px 12px; align-items: center; gap: 8px; z-index: 61; }
:deep(.bar-hint) { font-size: 13px; color: var(--text-secondary, #86868b); }
:deep(.bar-input) { width: 240px; padding: 6px 12px; border: 1px solid var(--border, #e5e5ea); border-radius: 8px; font-size: 13px; outline: none; }
:deep(.bar-btn) { height: 32px; padding: 0 10px; border-radius: 8px; border: 1px solid var(--border, #e5e5ea); background: var(--bg-secondary, #fff); display: flex; align-items: center; justify-content: center; cursor: pointer; font-size: 13px; white-space: nowrap; }
:deep(.bar-btn.primary) { background: var(--accent, #007aff); color: #fff; border: none; }
:deep(.bar-btn.danger) { color: #ff3b30; }
:deep(.bar-btn.success) { background: #34c759; color: #fff; border: none; }
.pulse-ring { position: absolute; border-radius: 8px; border: 2px solid var(--accent, #007aff); animation: pulseAnim 1s ease-out infinite; pointer-events: none; z-index: 62; }
@keyframes pulseAnim { 0%{opacity:1;transform:scale(1);} 100%{opacity:0;transform:scale(1.15);} }
.click-pointer { position: absolute; z-index: 63; pointer-events: none; font-size: 28px; animation: pointBounce 0.8s ease-in-out infinite; filter: drop-shadow(0 2px 4px rgba(0,0,0,.3)); }
@keyframes pointBounce { 0%,100%{transform:translateY(0) rotate(-15deg);} 50%{transform:translateY(-8px) rotate(-15deg);} }
</style>
