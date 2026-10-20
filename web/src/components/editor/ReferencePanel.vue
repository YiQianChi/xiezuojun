<template>
  <div class="reference-panel">
    <div class="ref-header">
      <span class="ref-title">📖 参考文档</span>
      <button class="ref-close" @click="$emit('close')" title="关闭">✕</button>
    </div>
    <div class="ref-body">
      <select v-model="selectedId" class="ref-select" @change="loadContent">
        <option :value="null">— 选择文档 —</option>
        <option v-for="doc in docOptions" :key="doc.id" :value="doc.id">{{ doc.title }}</option>
      </select>
      <div v-if="loading" class="ref-loading">加载中...</div>
      <div v-else-if="content !== null" class="ref-content prose-view" v-html="content"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { documentApi } from '../../core/api'

const props = defineProps<{
  documents: any[]
  projectId: string | number
}>()

defineEmits<{ close: [] }>()

const selectedId = ref<number | null>(null)
const content = ref<string | null>(null)
const loading = ref(false)

// ponytail: 排除文件夹类型，只列可读文档
const folderTypes = ['volume', 'outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline']
const docOptions = computed(() =>
  props.documents.filter(d => !folderTypes.includes(d.type))
)

async function loadContent() {
  if (!selectedId.value) {
    content.value = null
    return
  }
  loading.value = true
  try {
    const res = await documentApi.get(selectedId.value)
    content.value = res.data?.data?.content || res.data?.content || '(空文档)'
  } catch {
    content.value = '加载失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.reference-panel {
  width: 300px;
  flex-shrink: 0;
  border-left: 1px solid var(--border);
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ref-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.ref-close {
  border: none;
  background: none;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-tertiary);
  padding: 2px 6px;
  border-radius: var(--radius-xs);
}
.ref-close:hover {
  background: var(--accent-light);
  color: var(--accent);
}
.ref-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ref-select {
  margin: 8px;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 13px;
}
.ref-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-primary);
}
.ref-loading {
  padding: 16px;
  text-align: center;
  color: var(--text-tertiary);
}
/* ponytail: v-html needs scoped deep styles to reach injected nodes */
.prose-view :deep(h1) { font-size: 1.4em; font-weight: 700; margin: 0.6em 0 0.3em; }
.prose-view :deep(h2) { font-size: 1.2em; font-weight: 600; margin: 0.6em 0 0.3em; }
.prose-view :deep(h3) { font-size: 1.1em; font-weight: 600; margin: 0.5em 0 0.3em; }
.prose-view :deep(p) { margin: 0.4em 0; }
.prose-view :deep(blockquote) {
  border-left: 3px solid var(--border);
  padding-left: 12px;
  margin: 0.5em 0;
  color: var(--text-secondary);
}
.prose-view :deep(ul), .prose-view :deep(ol) { padding-left: 1.5em; margin: 0.4em 0; }
.prose-view :deep(li) { margin: 0.2em 0; }
.prose-view :deep(strong) { font-weight: 600; }
.prose-view :deep(em) { font-style: italic; }
.prose-view :deep(s) { text-decoration: line-through; color: var(--text-tertiary); }
.prose-view :deep(mark) { background: rgba(255,235,59,0.35); border-radius: 2px; padding: 0 2px; }
.prose-view :deep(table) { border-collapse: collapse; width: 100%; margin: 0.5em 0; }
.prose-view :deep(th), .prose-view :deep(td) { border: 1px solid var(--border); padding: 4px 8px; font-size: 0.9em; }
.prose-view :deep(code) {
  background: rgba(0,0,0,0.05);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 0.9em;
}
</style>
