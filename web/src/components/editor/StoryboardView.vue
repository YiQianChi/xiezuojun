<template>
  <div class="storyboard">
    <div v-if="chapters.length === 0" class="sb-empty">
      还没有章节，去写作模式添加章节吧
    </div>
    <div v-else class="sb-track">
      <div
        v-for="ch in chapters"
        :key="ch.id"
        class="sb-card"
        :class="{ active: ch.id === activeDocId, dragging: dragId === ch.id }"
        draggable="true"
        @click="emit('select-doc', ch)"
        @dragstart="onDragStart($event, ch.id)"
        @dragend="dragId = null"
        @dragover.prevent="onDragOver($event, ch.id)"
        @drop.prevent="onDrop(ch.id)"
      >
        <span class="sb-dot" :style="{ background: statusColor[ch.doc_status] ?? '#8e8e93' }" />
        <span class="sb-title">{{ ch.title }}</span>
        <span class="sb-words">{{ formatWords(ch.word_count) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  documents: any[]
  activeDocId: number | null
}>()

const emit = defineEmits<{
  'select-doc': [doc: any]
  'reorder': [payload: { fromId: number; toId: number; position: 'before' | 'after' }]
}>()

const chapters = computed(() =>
  props.documents
    .filter(d => d.type === 'chapter')
    .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0))
)

const statusColor: Record<string, string> = {
  draft: '#8e8e93',
  revising: '#ff9500',
  final: '#34c759',
}

const dragId = ref<number | null>(null)
const dropBefore = ref(true)

function onDragStart(e: DragEvent, id: number) {
  dragId.value = id
  e.dataTransfer!.effectAllowed = 'move'
}

function onDragOver(e: DragEvent, targetId: number) {
  if (targetId === dragId.value) return
  // ponytail: half-width threshold → before/after, no drop indicator line
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  dropBefore.value = e.clientX < rect.left + rect.width / 2
  e.dataTransfer!.dropEffect = 'move'
}

function onDrop(targetId: number) {
  if (dragId.value !== null && dragId.value !== targetId) {
    emit('reorder', {
      fromId: dragId.value,
      toId: targetId,
      position: dropBefore.value ? 'before' : 'after',
    })
  }
  dragId.value = null
}

function formatWords(n?: number) {
  if (!n) return '0字'
  return n >= 1000 ? `${(n / 1000).toFixed(1)}k字` : `${n}字`
}
</script>

<style scoped>
.storyboard {
  background: var(--bg-secondary, #f5f5f7);
  padding: 12px;
  overflow-x: auto;
}
.sb-track {
  display: flex;
  gap: 10px;
  min-height: 96px;
}
.sb-empty {
  color: var(--text-tertiary);
  font-size: 13px;
  text-align: center;
  padding: 32px 0;
}
.sb-card {
  flex-shrink: 0;
  width: 150px;
  background: var(--bg-primary, #fff);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  padding: 10px 12px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border: 2px solid transparent;
  transition: border-color 0.15s ease, opacity 0.15s ease;
}
.sb-card:hover { border-color: rgba(0, 122, 255, 0.25); }
.sb-card.active { border-color: var(--accent, #007aff); }
.sb-card.dragging { opacity: 0.4; }
.sb-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.sb-title {
  font-family: var(--font-serif);
  font-size: 14px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}
.sb-words {
  font-size: 11px;
  color: var(--text-tertiary);
}
</style>
