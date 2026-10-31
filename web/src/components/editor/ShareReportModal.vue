<template>
  <Teleport to="body">
    <div v-if="open" class="report-overlay" @click.self="$emit('update:open', false)">
      <div class="report-modal">
        <div class="rm-head">
          <span class="rm-title">📢 今日战报</span>
          <button class="rm-close" @click="$emit('update:open', false)">✕</button>
        </div>
        <div v-if="loading" class="rm-loading">生成中…</div>
        <template v-else>
          <canvas ref="canvasRef" class="rm-canvas" :width="750" :height="1100"></canvas>
          <div class="rm-tip">长按或右键保存图片 · 发朋友圈晒一晒</div>
          <button class="rm-save" @click="download">⬇ 保存图片</button>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { buddyApi } from '../../core/api'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'update:open', v: boolean): void }>()

interface ReportData {
  date: string
  today_words: number
  today_min: number
  rate: string
  streak_days: number
  trend: { date: string; words: number }[]
}

const loading = ref(false)
const canvasRef = ref<HTMLCanvasElement | null>(null)
let report: ReportData | null = null

watch(() => props.open, async (v) => {
  if (!v) return
  loading.value = true
  try {
    const res = await buddyApi.report()
    report = res.data?.data ?? null
  } catch { report = null }
  loading.value = false
  if (report) {
    await nextTick()
    draw()
  }
})

// ---- 品牌绿主题（#81bb8e 系）Canvas 绘卡 750×1100（3:4.4） ----
function draw() {
  const cv = canvasRef.value
  if (!cv || !report) return
  const ctx = cv.getContext('2d')
  if (!ctx) return
  const W = 750, H = 1100
  const d = report

  // 背景：纸感渐变
  const bg = ctx.createLinearGradient(0, 0, W * 0.4, H)
  bg.addColorStop(0, '#f6faf7')
  bg.addColorStop(0.55, '#e8f3ea')
  bg.addColorStop(1, '#dcefe1')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, H)
  // 顶部品牌绿色带
  ctx.fillStyle = '#81bb8e'
  ctx.fillRect(0, 0, W, 10)

  // 日期
  ctx.fillStyle = '#7ba589'
  ctx.font = '500 30px "PingFang SC", sans-serif'
  ctx.fillText(d.date, 60, 110)

  // 大字：今日字数
  ctx.fillStyle = '#3f7a52'
  ctx.font = '800 150px Georgia, "PingFang SC", serif'
  const numStr = d.today_words.toLocaleString()
  const numW = ctx.measureText(numStr).width
  ctx.fillText(numStr, 60, 270)
  ctx.font = '600 44px "PingFang SC", sans-serif'
  ctx.fillText('字', 70 + numW, 270)

  // 连续天数
  ctx.fillStyle = '#5c9670'
  ctx.font = '500 34px "PingFang SC", sans-serif'
  ctx.fillText(`🔥 连续写作 ${d.streak_days} 天`, 60, 340)

  // 两枚统计卡
  const cards = [
    { v: fmtDur(d.today_min), l: '码字时长' },
    { v: d.rate ? d.rate : '—', l: '字 / 分钟' },
  ]
  cards.forEach((c, i) => {
    const x = 60 + i * 320, y = 390, w = 300, h = 150
    ctx.fillStyle = 'rgba(255,255,255,0.75)'
    roundRect(ctx, x, y, w, h, 24)
    ctx.fillStyle = '#3f7a52'
    ctx.font = '700 54px "PingFang SC", sans-serif'
    ctx.fillText(c.v, x + w / 2 - ctx.measureText(c.v).width / 2, y + 88)
    ctx.fillStyle = '#8aab97'
    ctx.font = '400 28px "PingFang SC", sans-serif'
    ctx.fillText(c.l, x + w / 2 - ctx.measureText(c.l).width / 2, y + 128)
  })

  // 迷你趋势线（近14天）
  const tx = 60, ty = 610, tw = W - 120, th = 150
  const vals = d.trend.map((p) => p.words)
  const max = Math.max(1, ...vals)
  const px = (i: number) => tx + (tw * i) / (vals.length - 1)
  const py = (v: number) => ty + th - (v / max) * (th - 16)
  ctx.strokeStyle = '#5c9670'
  ctx.lineWidth = 5
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  ctx.beginPath()
  vals.forEach((v, i) => (i ? ctx.lineTo(px(i), py(v)) : ctx.moveTo(px(i), py(v))))
  ctx.stroke()
  // 端点
  ctx.fillStyle = '#3f7a52'
  ctx.beginPath()
  ctx.arc(px(vals.length - 1), py(vals[vals.length - 1]), 9, 0, Math.PI * 2)
  ctx.fill()
  ctx.fillStyle = '#8aab97'
  ctx.font = '400 26px "PingFang SC", sans-serif'
  const cap = '近 14 天字数趋势'
  ctx.fillText(cap, tx + tw / 2 - ctx.measureText(cap).width / 2, ty + th + 46)

  // 底部品牌行
  const by = H - 130
  ctx.strokeStyle = '#bcd8c4'
  ctx.setLineDash([8, 10])
  ctx.beginPath(); ctx.moveTo(60, by); ctx.lineTo(W - 60, by); ctx.stroke()
  ctx.setLineDash([])
  ctx.fillStyle = '#81bb8e'
  roundRect(ctx, 60, by + 32, 58, 58, 14)
  ctx.fillStyle = '#fff'
  ctx.font = '700 32px "PingFang SC", sans-serif'
  ctx.fillText('菌', 76, by + 72)
  ctx.fillStyle = '#5c9670'
  ctx.font = '500 30px "PingFang SC", sans-serif'
  ctx.fillText('写作菌 · 一个写故事的地方', 140, by + 70)
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
  ctx.fill()
}

function fmtDur(min: number): string {
  if (!min) return '0m'
  return min >= 60 ? `${Math.floor(min / 60)}h${min % 60 ? ' ' + (min % 60) + 'm' : ''}` : `${min}m`
}

function download() {
  const cv = canvasRef.value
  if (!cv) return
  const a = document.createElement('a')
  a.href = cv.toDataURL('image/png')
  a.download = `写作战报-${new Date().toISOString().slice(0, 10)}.png`
  a.click()
}
</script>

<style scoped>
.report-overlay {
  position: fixed; inset: 0; z-index: 1200;
  background: rgba(0,0,0,0.45);
  display: flex; align-items: center; justify-content: center;
}
.report-modal {
  /* 色彩统一 v1.2：Teleport 弹层带一套暖米变量（同搭子窗做法） */
  --bg-secondary: #f8f3e8;
  --text-secondary: #7a7263;
  --text-tertiary: #a39a86;
  background: #f8f3e8;
  border: 1px solid #e2d8c2;
  border-radius: 20px; padding: 18px;
  box-shadow: 0 20px 60px rgba(60, 45, 20, 0.28);
  max-width: 92vw; max-height: 92dvh;
  display: flex; flex-direction: column; align-items: center;
}
.rm-head { display: flex; align-items: center; justify-content: space-between; width: 100%; margin-bottom: 12px; }
.rm-title { font-size: 15px; font-weight: 700; color: var(--text-primary, #333); }
.rm-close { border: none; background: transparent; font-size: 16px; cursor: pointer; color: var(--text-tertiary, #999); }
.rm-loading { padding: 60px 80px; color: var(--text-tertiary, #999); font-size: 14px; }
.rm-canvas { width: min(375px, 80vw); height: auto; border-radius: 14px; box-shadow: 0 4px 24px rgba(0,0,0,0.12); }
.rm-tip { font-size: 11.5px; color: var(--text-tertiary, #999); margin: 10px 0; }
.rm-save {
  border: none; cursor: pointer; border-radius: 12px; padding: 10px 36px;
  background: linear-gradient(135deg, #8fc59b, #6aa877); color: #fff;
  font-size: 14px; font-weight: 600;
  box-shadow: 0 4px 14px rgba(129,187,142,0.4);
}
.rm-save:active { transform: scale(0.97); }
</style>
