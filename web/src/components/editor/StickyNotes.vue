<template>
  <!-- 便签贴纸层：贴在正文纸张右缘（C 方案）。对齐方式=实时量 .ProseMirror 纸张右缘（AI 面板开合纸张会移）。 -->
  <div v-if="notes.length || draftOpen" class="sticky-layer" ref="layerEl" :style="layerStyle">
    <div
      v-for="(n, i) in shownNotes"
      :key="n.id"
      class="sk-note"
      :class="[n.color, { open: editingId === n.id }]"
      :style="{ top: 96 + i * 104 + 'px' }"
      @click="editingId === n.id ? null : (editingId = n.id)"
    >
      <template v-if="editingId === n.id">
        <textarea v-model="editDraft" rows="4" placeholder="写点备忘…" @keydown.esc="editingId = null" @click.stop></textarea>
        <div class="sk-tools">
          <button v-for="c in ['y', 'b', 'g']" :key="c" class="sk-dot" :class="[c, { on: editColor === c }]" @click.stop="editColor = c"></button>
          <button class="sk-save" @click.stop="saveEdit(n)">保存</button>
          <button class="sk-del" title="撕掉" @click.stop="removeNote(n)">🗑</button>
        </div>
      </template>
      <template v-else>
        <div class="sk-t">{{ n.content }}</div>
        <span class="sk-x" title="撕掉" @click.stop="removeNote(n)">✕</span>
      </template>
    </div>
    <button class="sk-add" title="贴一张新便签" @click="openDraft">＋</button>

    <!-- 新建：贴纸形态的输入卡 -->
    <div v-if="draftOpen" class="sk-note y open draft" :style="{ top: 96 + shownNotes.length * 104 + 'px' }">
      <textarea v-model="draftText" ref="draftTa" rows="3" placeholder="想到就记…" @keydown.enter.exact.prevent="saveDraft" @keydown.esc="draftOpen = false"></textarea>
      <div class="sk-tools">
        <button v-for="c in ['y', 'b', 'g']" :key="c" class="sk-dot" :class="[c, { on: draftColor === c }]" @click.stop="draftColor = c"></button>
        <button class="sk-save" @click.stop="saveDraft">贴上</button>
        <button class="sk-del" @click.stop="draftOpen = false">✕</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { stickyApi } from '../../core/api'
import { toast } from '../../composables/useToast'

const props = defineProps<{ projectId: number }>()

interface Note { id: number; content: string; color: string }

const notes = ref<Note[]>([])
const editingId = ref<number | null>(null)
const editDraft = ref('')
const editColor = ref('y')
const draftOpen = ref(false)
const draftText = ref('')
const draftColor = ref('y')
const draftTa = ref<HTMLTextAreaElement | null>(null)
const layerEl = ref<HTMLElement | null>(null)
const layerStyle = ref<Record<string, string>>({})

const MAX_SHOWN = 6
const shownNotes = computed(() => notes.value.slice(0, MAX_SHOWN))

async function load() {
  if (!props.projectId) return
  try {
    const res = await stickyApi.list(props.projectId)
    const all = res.data?.data || []
    notes.value = all.filter((n: any) => n.type === 'note')
  } catch { notes.value = [] }
}
watch(() => props.projectId, () => { load(); editingId.value = null }, { immediate: true })

function openDraft() {
  draftOpen.value = true
  nextTick(() => draftTa.value?.focus())
}
async function saveDraft() {
  const v = draftText.value.trim()
  if (!v || !props.projectId) return
  try {
    const res = await stickyApi.create({ project_id: props.projectId, type: 'note', content: v, color: draftColor.value })
    notes.value.unshift(res.data?.data)
    draftText.value = ''
    draftOpen.value = false
  } catch { toast.error('保存失败') }
}
async function saveEdit(n: Note) {
  const v = editDraft.value.trim()
  editingId.value = null
  if (!v || v === n.content && editColor.value === n.color) return
  n.content = v
  n.color = editColor.value
  try { await stickyApi.update(n.id, { content: v, color: editColor.value }) } catch { toast.error('保存失败'); load() }
}
async function removeNote(n: Note) {
  notes.value = notes.value.filter(x => x.id !== n.id)
  if (editingId.value === n.id) editingId.value = null
  try { await stickyApi.remove(n.id) } catch { /* 下次进来对齐 */ }
}

// 点贴纸进入编辑态时预填
watch(editingId, (id) => {
  const n = notes.value.find(x => x.id === id)
  if (n) { editDraft.value = n.content; editColor.value = n.color || 'y' }
})

/* 对齐纸张右缘：量 .ProseMirror 的右边缘相对本层宿主（editor-main）的距离，1.5s 兜底轮询（AI 面板开合/窗口变化都会动） */
let measureTimer: ReturnType<typeof setInterval> | null = null
function measure() {
  const host = layerEl.value?.parentElement
  const paper = document.querySelector('.tiptap-container .ProseMirror')
  if (!host) return
  if (!paper) { layerStyle.value = { right: '24px' }; return }
  const pr = paper.getBoundingClientRect()
  const hr = host.getBoundingClientRect()
  const gap = hr.right - pr.right
  layerStyle.value = { right: Math.max(10, gap - 14) + 'px' }
}
onMounted(() => {
  measure()
  measureTimer = setInterval(measure, 1500)
  window.addEventListener('resize', measure)
})
onBeforeUnmount(() => {
  if (measureTimer) clearInterval(measureTimer)
  window.removeEventListener('resize', measure)
})
</script>

<style scoped>
.sticky-layer { position: absolute; top: 0; bottom: 0; z-index: 40; pointer-events: none; }
.sk-note {
  position: absolute; right: 0;
  width: 148px;
  border-radius: 10px 4px 4px 10px;
  padding: 9px 11px 11px;
  font-size: 11px; line-height: 1.55;
  cursor: pointer;
  box-shadow: 0 3px 10px rgba(90, 70, 20, 0.22);
  transition: transform 0.18s ease, width 0.18s ease;
  pointer-events: auto;
  word-break: break-all;
}
.sk-note::before {
  content: ''; position: absolute; top: -5px; left: 14px; width: 34px; height: 11px;
  background: rgba(255, 255, 255, 0.55); transform: rotate(-2deg);
  border-radius: 2px; box-shadow: 0 1px 2px rgba(0,0,0,0.08);
}
.sk-note:hover { transform: translateX(-7px); }
.sk-note.open { width: 200px; transform: translateX(-34px); z-index: 9; cursor: default; }
.sk-note.y { background: linear-gradient(180deg, #fdf6cf, #fbeeb2); color: #6b5a20; border: 1px solid #eedb9c; }
.sk-note.b { background: linear-gradient(180deg, #dcefff, #cfe6fa); color: #1e4a73; border: 1px solid #bcd9f2; }
.sk-note.g { background: linear-gradient(180deg, #ddf3e0, #cfead4); color: #2d5a3a; border: 1px solid #b8dfc0; }
.sk-t { max-height: 66px; overflow: hidden; }
.sk-x { position: absolute; top: 4px; right: 7px; font-size: 9px; opacity: 0.45; cursor: pointer; }
.sk-x:hover { opacity: 1; }
.sk-note textarea {
  width: 100%; border: none; background: rgba(255,255,255,0.5); border-radius: 6px;
  font-size: 11px; line-height: 1.55; padding: 6px 7px; color: inherit; outline: none; resize: none;
  font-family: inherit;
}
.sk-tools { display: flex; align-items: center; gap: 5px; margin-top: 6px; }
.sk-dot { width: 13px; height: 13px; border-radius: 50%; border: 1.5px solid rgba(0,0,0,0.15); cursor: pointer; padding: 0; }
.sk-dot.y { background: #fdf3c0; } .sk-dot.b { background: #d7ecff; } .sk-dot.g { background: #dcf2df; }
.sk-dot.on { border-color: currentColor; box-shadow: 0 0 0 2px rgba(255,255,255,0.7); }
.sk-save { border: none; border-radius: 6px; padding: 3px 9px; font-size: 10px; font-weight: 800; background: rgba(255,255,255,0.55); color: inherit; cursor: pointer; margin-left: auto; }
.sk-del { border: none; background: none; font-size: 11px; opacity: 0.6; cursor: pointer; padding: 0; }
.sk-del:hover { opacity: 1; }
.sk-add {
  position: absolute; right: -6px; top: 720px;
  width: 28px; height: 28px; border-radius: 50%;
  border: 1.5px dashed #c9b96a; color: #a8944a; background: rgba(253, 243, 192, 0.75);
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; cursor: pointer; pointer-events: auto;
}
.sk-add:hover { background: #fdf3c0; }
/* 手机端：贴纸收成右缘小色条（点开弹层编辑）——窄屏纸张近全宽，整张贴纸压字 */
@media (max-width: 768px) {
  .sticky-layer { display: none; }
}
</style>
