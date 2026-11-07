<template>
  <!-- 键盘上方标点条：只在手机键盘弹起且正文编辑器聚焦时出现——占键盘的地盘，不压正文 -->
  <div v-if="show" class="punct-bar" ref="barRef">
    <!-- 撤销/重做：手机没有 Ctrl+Z，删多了就地撤回 -->
    <button class="pb-hist" title="撤销" @mousedown.prevent @click="hist('undo')">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 14L4 9l5-5"/><path d="M4 9h10a6 6 0 016 6v1"/></svg>
    </button>
    <button class="pb-hist" title="重做" @mousedown.prevent @click="hist('redo')">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M15 14l5-5-5-5"/><path d="M20 9H10a6 6 0 00-6 6v1"/></svg>
    </button>
    <span class="pb-sep"></span>
    <button
      v-for="(s, i) in symbols" :key="i"
      @mousedown.prevent @click="tap(s)"
    >{{ s }}</button>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { toast } from '../../composables/useToast'

const props = defineProps<{ show: boolean; editor: any; baseline?: number }>()

// 写小说高频标点：成对引号写前面（选中文字点它=包住选区）
const symbols = ['「」', '『』', '“”', '……', '——', '、', '。', '！', '？', '；', '：']
const PAIRS = ['「」', '『』', '“”']

const barRef = ref<HTMLElement | null>(null)

function tap(s: string) {
  const ed = props.editor
  if (!ed) return
  const { from, to } = ed.state.selection
  if (PAIRS.includes(s)) {
    if (from < to) {
      // 有选区：开闭引号包住选中文字
      const selected = ed.state.doc.textBetween(from, to, '\n')
      ed.chain().focus().insertContentAt({ from, to }, s[0] + selected + s[1]).run()
    } else {
      // 无选区：插入成对引号，光标落在中间
      ed.chain().focus().insertContent(s).run()
      ed.chain().setTextSelection(ed.state.selection.from - 1).run()
    }
  } else {
    ed.chain().focus().insertContent(s).run()
  }
}

function hist(cmd: 'undo' | 'redo') {
  const ed = props.editor
  if (!ed) return
  if (cmd === 'undo') {
    const words = ed.getText().replace(/\n/g, '').length
    const floor = props.baseline ?? 0
    if (words <= floor) {
      // 地板：已回到本章打开时的内容，再往前不撤——防连点把全文撤没（撤没还会被自动保存存走）
      toast.info('已回到本章打开时的内容')
      return
    }
    ed.chain().focus().undo().run()
    if (ed.getText().replace(/\n/g, '').length < floor) {
      // 这一步撤过了头（成组跨越地板）：退回来守住地板
      ed.chain().redo().run()
      toast.info('已回到本章打开时的内容')
    }
  } else {
    ed.chain().focus().redo().run()
  }
}

/* 贴键盘顶：fixed 元素锚的是布局视口，iOS 键盘弹起布局视口不变、可视视口缩——
   用 visualViewport 的 height+offsetTop 算可视底缘；安卓 adjustResize 布局视口自己缩，公式同样成立 */
let cleanup = () => {}
onMounted(() => {
  const vv = window.visualViewport
  if (!vv) return
  const place = () => {
    const el = barRef.value
    if (el) el.style.top = (vv.height + vv.offsetTop - 44) + 'px'
  }
  vv.addEventListener('resize', place)
  vv.addEventListener('scroll', place)
  const timer = setInterval(place, 250) // 键盘升起的动画期连续贴住
  cleanup = () => {
    vv.removeEventListener('resize', place)
    vv.removeEventListener('scroll', place)
    clearInterval(timer)
  }
})
onBeforeUnmount(cleanup)
</script>

<style scoped>
.punct-bar {
  position: fixed;
  left: 0;
  right: 0;
  height: 44px;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 6px;
  overflow-x: auto;
  background: rgba(244, 240, 230, 0.97);
  border-top: 1px solid rgba(0, 0, 0, 0.08);
  z-index: 320;
  -webkit-overflow-scrolling: touch;
}
.punct-bar button {
  min-width: 40px;
  height: 36px;
  flex-shrink: 0;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 17px;
  color: #3a3a3a;
  cursor: pointer;
  padding: 0;
}
.punct-bar button:active { background: rgba(0, 0, 0, 0.07); }
.pb-hist { color: #6aa877 !important; min-width: 34px !important; }
.pb-sep { width: 1px; height: 22px; background: rgba(0, 0, 0, 0.12); flex-shrink: 0; margin: 0 4px; }
</style>
