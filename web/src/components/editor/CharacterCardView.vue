<template>
  <div class="char-card">
    <!-- 头部：头像 + 名字 + 角色徽标 + 一句话简介（全部就地编辑） -->
    <div class="cc-head">
      <div class="cc-ava" :title="'点击换头像'" @click="cycleAvatar">{{ meta.avatar || '友' }}</div>
      <div class="cc-title-wrap">
        <div class="cc-name" contenteditable spellcheck="false" @blur="onNameBlur" ref="nameEl">{{ docTitle }}</div>
        <div class="cc-role-row">
          <span class="cc-role" :style="{ background: roleStyle.bg }" title="点击切换角色定位" @click="cycleRole">{{ roleStyle.label }}</span>
        </div>
        <div class="cc-sub" contenteditable :data-ph="'一句话定位，如：余杭镇客栈伙计 · 16岁'" @blur="onFieldBlur('desc', $event)">{{ meta.desc }}</div>
      </div>
    </div>

    <!-- 字段区：身份 / 标签 / 关系 -->
    <div class="cc-fields">
      <div class="cc-row" v-if="meta.identity">
        <span class="cc-label">身份</span>
        <div class="cc-input" contenteditable :data-ph="'点这里写身份'" @blur="onFieldBlur('identity', $event)">{{ meta.identity }}</div>
      </div>
      <div class="cc-row" v-else>
        <span class="cc-label"></span>
        <div class="cc-input cc-input-ghost" contenteditable :data-ph="'＋ 身份（可选）'" @blur="onFieldBlur('identity', $event)"></div>
      </div>
      <div class="cc-row">
        <span class="cc-label">标签</span>
        <div class="cc-tags">
          <span v-for="(t, i) in meta.tags" :key="i" class="cc-tag">{{ t }}<span class="tx" @click="removeTag(i)">×</span></span>
          <button v-if="!tagAdding" class="cc-tag-add" @click="startTagAdd">＋</button>
          <span v-else class="cc-tag cc-tag-input" contenteditable ref="tagInputEl" @keydown.enter.prevent="commitTag" @blur="commitTag"></span>
        </div>
      </div>
      <div class="cc-row">
        <span class="cc-label">关系</span>
        <div class="cc-rels">
          <div v-for="(r, i) in meta.rels" :key="i" class="cc-rel">
            <span class="cc-rel-name" contenteditable spellcheck="false" @blur="onRelBlur(i, 'name', $event)">{{ r.name }}</span>
            <span class="cc-rel-desc" contenteditable :data-ph="'关系…'" @blur="onRelBlur(i, 'desc', $event)">{{ r.desc }}</span>
            <span class="tx" @click="removeRel(i)">×</span>
          </div>
          <div class="cc-rel-add" @click="addRel">＋ 关系</div>
        </div>
      </div>
    </div>

    <!-- 小传：就是文档正文，走同一个编辑器（demo 形态：干净无按钮） -->
    <div class="cc-body">
      <div class="cc-body-label">小传</div>
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'

const props = defineProps<{
  doc: any // character 类型文档（含 meta 字符串）
  editor?: any // 父组件传入的 tiptap 编辑器实例（小节插入用 tiptap 命令，绕开 DOM 模拟）
}>()
const emit = defineEmits<{
  (e: 'update-meta', meta: Record<string, any>): void
  (e: 'update-title', title: string): void
}>()

// meta 解析 + 本地可变副本（就地编辑零散提交，最后统一 emit）
const meta = computed<Record<string, any>>(() => {
  const base: Record<string, any> = { role: '', desc: '', identity: '', tags: [], rels: [], avatar: '' }
  if (!props.doc?.meta) return base
  try {
    const m = JSON.parse(props.doc.meta)
    return { ...base, ...m, tags: m.tags || [], rels: m.rels || [] }
  } catch { return base }
})
const docTitle = computed(() => props.doc?.title || '')

function pushMeta(patch: Record<string, any>) {
  emit('update-meta', { ...meta.value, ...patch })
}

// ---- 名字（同步 document.title） ----
function onNameBlur(e: FocusEvent) {
  const t = (e.target as HTMLElement).textContent?.trim() || ''
  if (t && t !== props.doc.title) emit('update-title', t)
}

// ---- 头像：点击在 首字 ↔ 常用 emoji 间轮换 ----
const AVA_EMOJIS = ['🧑', '👩', '🧓', '👨‍🦱', '👦', '👧', '🧝', '🧙', '⚔️', '🌙', '🔥', '🌿']
function cycleAvatar() {
  const cur = meta.value.avatar || ''
  let next: string
  if (cur === AVA_EMOJIS[AVA_EMOJIS.length - 1] || !cur) {
    next = firstRune(docTitle.value) // 轮回：emoji 尾 → 回首字
  } else {
    const i = AVA_EMOJIS.indexOf(cur)
    next = i >= 0 ? AVA_EMOJIS[(i + 1) % AVA_EMOJIS.length] : firstRune(docTitle.value)
    if (AVA_EMOJIS.indexOf(next) === 0 && i === -1) { /* 首字 → 第一个 emoji */ }
  }
  if (!cur) next = firstRune(docTitle.value)
  pushMeta({ avatar: next })
}
function firstRune(s: string): string {
  for (const r of s || '') return r
  return '友'
}

// ---- 角色定位轮换 ----
const ROLES = [
  { key: 'main', label: '主角', bg: '#81bb8e' },
  { key: 'support', label: '配角', bg: '#8fa9c9' },
  { key: 'villain', label: '反派', bg: '#d98a7a' },
  { key: 'extra', label: '路人', bg: '#b3bfb5' },
]
const roleStyle = computed(() => ROLES.find(r => r.key === (meta.value.role || 'main')) || ROLES[0])
function cycleRole() {
  const i = ROLES.findIndex(r => r.key === (meta.value.role || 'main'))
  pushMeta({ role: ROLES[(i + 1) % ROLES.length].key })
}

// ---- 通用字段就地编辑（desc/identity） ----
function onFieldBlur(field: string, e: FocusEvent) {
  pushMeta({ [field]: (e.target as HTMLElement).textContent?.trim() || '' })
}

// ---- 标签 ----
const tagAdding = ref(false)
const tagInputEl = ref<HTMLElement | null>(null)
function startTagAdd() {
  tagAdding.value = true
  nextTick(() => {
    tagInputEl.value?.focus()
  })
}
function commitTag() {
  const el = tagInputEl.value
  if (!el) return
  const t = el.textContent?.trim()
  if (t) pushMeta({ tags: [...meta.value.tags, t] })
  tagAdding.value = false
}
function removeTag(i: any) {
  pushMeta({ tags: meta.value.tags.filter((_: any, x: number) => x !== i) })
}

// ---- 关系 ----
function addRel() {
  pushMeta({ rels: [...meta.value.rels, { name: '新角色', desc: '关系' }] })
}
function removeRel(i: any) {
  pushMeta({ rels: meta.value.rels.filter((_: any, x: number) => x !== i) })
}
function onRelBlur(i: any, field: 'name' | 'desc', e: FocusEvent) {
  const v = (e.target as HTMLElement).textContent?.trim() || ''
  const rels = meta.value.rels.map((r: any, x: number) => x === i ? { ...r, [field]: v } : r)
  pushMeta({ rels })
}
</script>

<style scoped>
.char-card { padding: 4px 0 12px; }
.cc-head { display: flex; align-items: flex-start; gap: 14px; padding: 8px 4px 14px; }
.cc-ava {
  width: 56px; height: 56px; border-radius: 16px; flex-shrink: 0; cursor: pointer;
  background: linear-gradient(135deg, #a8ceb2, #7fb28d); color: #fff;
  font-size: 24px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  transition: filter .15s;
}
.cc-ava:hover { filter: brightness(.92); }
.cc-title-wrap { flex: 1; min-width: 0; }
.cc-name {
  font-size: 20px; font-weight: 700; color: var(--text-primary, #333);
  outline: none; border-radius: 6px; padding: 1px 5px; margin-left: -5px; cursor: text;
}
.cc-name:focus { background: #f4f8f5; box-shadow: 0 0 0 2px rgba(129,187,142,.35); }
.cc-role-row { margin-top: 4px; }
.cc-role { font-size: 10px; color: #fff; border-radius: 99px; padding: 2px 11px; cursor: pointer; user-select: none; }
.cc-sub {
  font-size: 12px; color: var(--text-tertiary, #98a89a); margin-top: 6px;
  outline: none; border-radius: 6px; padding: 1px 5px; margin-left: -5px; cursor: text; min-width: 200px;
}
.cc-sub:focus { background: #f4f8f5; box-shadow: 0 0 0 2px rgba(129,187,142,.35); }
.cc-sub:empty::before { content: attr(data-ph); color: #c3cfc5; }

.cc-fields { display: flex; flex-direction: column; gap: 10px; padding: 0 4px 14px; }
.cc-row { display: flex; align-items: flex-start; gap: 10px; font-size: 12.5px; }
.cc-label { color: var(--text-tertiary, #a8b5ab); font-size: 11.5px; width: 38px; flex-shrink: 0; padding-top: 4px; }
.cc-input {
  flex: 1; outline: none; border-radius: 6px; padding: 2px 5px; margin-left: -5px; cursor: text;
  color: var(--text-primary, #444); font-size: 13px;
}
.cc-input:focus { background: #f4f8f5; box-shadow: 0 0 0 2px rgba(129,187,142,.35); }
.cc-input:empty::before { content: attr(data-ph); color: #c3cfc5; }
.cc-input-ghost { font-size: 12px; color: #9ab9a2; }
.cc-tags { display: flex; gap: 6px; flex-wrap: wrap; }
.cc-tag {
  font-size: 11px; color: #5c9670; background: rgba(129,187,142,.13); border-radius: 99px;
  padding: 2px 10px; display: inline-flex; align-items: center; gap: 6px;
}
.cc-tag .tx { color: #b3c9b8; cursor: pointer; font-size: 10px; line-height: 1; }
.cc-tag .tx:hover { color: #d98a7a; }
.cc-tag-add {
  font-size: 11px; color: #9ab9a2; border: 1px dashed #bcd8c4; border-radius: 99px;
  padding: 1px 10px; cursor: pointer; background: transparent;
}
.cc-tag-add:hover { background: #f7fbf8; }
.cc-tag-input { min-width: 60px; outline: none; }
.cc-rels { display: flex; flex-direction: column; gap: 6px; flex: 1; }
.cc-rel { display: flex; align-items: center; gap: 8px; font-size: 12.5px; }
.cc-rel-name {
  color: #3f7a52; font-weight: 600; outline: none; border-radius: 5px; padding: 1px 5px;
  margin-left: -5px; cursor: text; min-width: 70px;
}
.cc-rel-name:focus { background: #f4f8f5; box-shadow: 0 0 0 2px rgba(129,187,142,.35); }
.cc-rel-desc {
  color: var(--text-tertiary, #8a9a8c); outline: none; border-radius: 5px; padding: 1px 5px;
  margin-left: -5px; cursor: text; flex: 1;
}
.cc-rel-desc:focus { background: #f4f8f5; box-shadow: 0 0 0 2px rgba(129,187,142,.35); }
.cc-rel-desc:empty::before { content: attr(data-ph); color: #c3cfc5; }
.cc-rel .tx { color: #c3cfc5; cursor: pointer; font-size: 11px; padding: 0 3px; }
.cc-rel .tx:hover { color: #d98a7a; }
.cc-rel-add { font-size: 11.5px; color: #9ab9a2; cursor: pointer; width: fit-content; padding: 2px 6px; border-radius: 6px; }
.cc-rel-add:hover { background: #f7fbf8; color: #5c9670; }

.cc-body { border-top: 1px solid var(--border-subtle, #f0f3f0); padding-top: 14px; }
/* 小传里的标题（## 性格 / ## 能力…）：绿色强调+足够对比（正文17px→标题23px），和卡片头部呼应 */
.cc-body :deep(.ProseMirror h2) {
  font-size: 1.35em;
  font-weight: 700;
  color: #3f7a52;
  margin: 1.1em 0 0.5em;
  padding-bottom: 0.25em;
  border-bottom: 1px solid rgba(129, 187, 142, 0.25);
}
.cc-body :deep(.ProseMirror h1) { display: none; } /* 卡头已有大名字，正文 h1 冗余（老卡里已存在的也不显） */
.cc-body-label { font-size: 11px; color: var(--text-tertiary, #a8b5ab); margin-bottom: 8px; display: flex; }
</style>
