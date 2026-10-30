<template>
  <Teleport to="body">
    <div v-if="open" class="phone-preview-overlay" @click="$emit('close')">
      <div class="phone-preview-drawer" @click.stop>
        <div class="pp-head">
          <b>📱 读者视角</b>
          <span class="pp-sub">正文实时同步 · 点外部关闭</span>
          <button class="pp-close" @click="$emit('close')" title="关闭">✕</button>
        </div>

        <div class="pp-ctrl">
          <div class="pp-seg">
            <span :class="{ on: theme === 'paper' }" @click="theme = 'paper'">纸白</span>
            <span :class="{ on: theme === 'green' }" @click="theme = 'green'">护眼绿</span>
            <span :class="{ on: theme === 'dark' }" @click="theme = 'dark'">夜间</span>
          </div>
          <div class="pp-font">
            <span @click="fontSize = Math.max(14, fontSize - 1)" title="缩小">A⁻</span>
            <span class="pp-font-val">{{ fontSize }}</span>
            <span @click="fontSize = Math.min(22, fontSize + 1)" title="放大">A⁺</span>
          </div>
        </div>

        <div class="pp-stage">
          <div class="pp-phone">
            <div class="pp-screen" :class="theme">
              <div class="pp-status"><span>9:41</span><span>📶 🔋</span></div>
              <div class="pp-rhead"><span>‹ 书架</span><b>{{ bookName }}</b><span>⇪</span></div>
              <div class="pp-rbody" ref="bodyRef">
                <div class="pp-rtitle">{{ title || '未命名章节' }}</div>
                <div class="pp-rmeta">{{ bookName }} · {{ wordCount }} 字</div>
                <p v-for="(para, i) in paragraphs" :key="i">{{ para }}</p>
                <div class="pp-end">— END · 下一章 —</div>
              </div>
              <div class="pp-rfoot">
                <span>上一章</span>
                <div class="pp-progress"><i :style="{ width: progressPct + '%' }"></i></div>
                <span>下一章</span>
              </div>
              <div class="pp-homebar"><i></i></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  open: boolean
  title: string
  bookName: string
  text: string
  wordCount: number
  chapterIndex: number
  chapterTotal: number
}>()

defineEmits<{ close: [] }>()

const theme = ref<'paper' | 'green' | 'dark'>('paper')
const fontSize = ref(17)
const bodyRef = ref<HTMLElement | null>(null)

const paragraphs = computed(() =>
  (props.text || '')
    .split('\n')
    .map(l => l.replace(/^#{1,6}\s*/, '').trim())
    .filter(Boolean)
)

const progressPct = computed(() => {
  if (!props.chapterTotal) return 30
  return Math.min(100, Math.round((props.chapterIndex / props.chapterTotal) * 100))
})

// 打开时回到顶部
watch(() => props.open, v => {
  if (v) setTimeout(() => { if (bodyRef.value) bodyRef.value.scrollTop = 0 }, 50)
})
</script>

<style scoped>
.phone-preview-overlay {
  position: fixed; inset: 0; z-index: 600;
  background: rgba(40, 32, 18, 0.35);
}
.phone-preview-drawer {
  position: absolute; top: 0; right: 0; bottom: 0;
  width: 440px; max-width: 92vw;
  background: var(--panel, #f8f3e8);
  border-left: 1px solid #e2d8c2;
  box-shadow: -12px 0 40px rgba(60, 45, 20, 0.18);
  display: flex; flex-direction: column;
  animation: pp-slide 0.22s ease;
}
@keyframes pp-slide { from { transform: translateX(40px); opacity: 0; } }

.pp-head {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 16px 10px; border-bottom: 1px solid rgba(0,0,0,0.06);
}
.pp-head b { font-size: 14px; }
.pp-sub { font-size: 11px; color: var(--text-tertiary, #aeaeb2); flex: 1; }
.pp-close {
  border: none; background: transparent; font-size: 14px;
  color: var(--text-tertiary, #86868b); cursor: pointer; padding: 4px 8px; border-radius: 6px;
}
.pp-close:hover { background: rgba(0,0,0,0.06); }

.pp-ctrl {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 16px;
}
.pp-seg { display: flex; gap: 4px; background: rgba(0,0,0,0.05); border-radius: 9px; padding: 3px; }
.pp-seg span {
  font-size: 12px; padding: 5px 12px; border-radius: 7px;
  color: var(--text-secondary, #86868b); cursor: pointer;
}
.pp-seg span.on {
  background: #fff; color: var(--accent, #6aa877); font-weight: 600;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}
.pp-font { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--text-secondary, #86868b); }
.pp-font span { cursor: pointer; padding: 2px 6px; border-radius: 5px; }
.pp-font span:hover { background: rgba(0,0,0,0.06); }
.pp-font-val { min-width: 20px; text-align: center; font-variant-numeric: tabular-nums; }

.pp-stage { flex: 1; display: flex; align-items: center; justify-content: center; padding: 0 16px 20px; min-height: 0; }
.pp-phone {
  width: 330px; max-width: 100%; height: 100%; max-height: 640px;
  border-radius: 36px; background: #1b1d22; padding: 10px;
  box-shadow: 0 16px 44px rgba(60,45,20,0.25), inset 0 0 0 2px #3a3d44;
  position: relative;
}
.pp-phone::before {
  content: ''; position: absolute; top: 16px; left: 50%; transform: translateX(-50%);
  width: 80px; height: 20px; border-radius: 12px; background: #1b1d22; z-index: 5;
}
.pp-screen {
  --r-bg: #f6f1e5; --r-ink: #3a362e;
  width: 100%; height: 100%; border-radius: 28px; overflow: hidden;
  display: flex; flex-direction: column;
  background: var(--r-bg); color: var(--r-ink);
  transition: background 0.3s, color 0.3s;
}
.pp-screen.green { --r-bg: #cfe0c8; --r-ink: #2e3b2b; }
.pp-screen.dark { --r-bg: #17181c; --r-ink: #b8b2a6; }

.pp-status {
  flex: 0 0 40px; display: flex; justify-content: space-between; align-items: flex-end;
  padding: 0 22px 4px; font-size: 12px; font-weight: 600;
}
.pp-rhead {
  flex: 0 0 36px; display: flex; align-items: center; justify-content: space-between;
  padding: 0 14px; font-size: 11px; opacity: 0.75;
}
.pp-rhead b { font-weight: 500; max-width: 55%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pp-rbody { flex: 1; overflow-y: auto; padding: 4px 18px 24px; }
.pp-rtitle {
  font-family: var(--font-serif, serif);
  font-size: 19px; font-weight: 700; text-align: center;
  margin: 12px 0 4px; letter-spacing: 1px;
}
.pp-rmeta { text-align: center; font-size: 10px; opacity: 0.55; margin-bottom: 14px; }
.pp-rbody p {
  font-family: var(--font-serif, serif);
  font-size: v-bind(fontSize + 'px');
  line-height: 1.9; text-indent: 2em; margin-bottom: 0.6em;
}
.pp-end { text-align: center; opacity: 0.5; font-size: 11px; padding: 22px 0 8px; }
.pp-rfoot {
  flex: 0 0 40px; display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px; font-size: 10px; opacity: 0.75;
  border-top: 1px solid rgba(0,0,0,0.06);
}
.pp-screen.dark .pp-rfoot { border-top-color: rgba(255,255,255,0.07); }
.pp-progress { flex: 1; height: 3px; border-radius: 2px; background: rgba(0,0,0,0.10); margin: 0 12px; overflow: hidden; }
.pp-screen.dark .pp-progress { background: rgba(255,255,255,0.12); }
.pp-progress i { display: block; height: 100%; background: var(--accent, #6aa877); }
.pp-homebar { flex: 0 0 18px; display: flex; justify-content: center; align-items: flex-end; padding-bottom: 5px; }
.pp-homebar i { width: 96px; height: 4px; border-radius: 2px; background: var(--r-ink); opacity: 0.4; }

@media (max-width: 768px) {
  .phone-preview-drawer { width: 100vw; max-width: 100vw; border-left: none; }
}
</style>
