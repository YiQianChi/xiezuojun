<template>
  <div class="memory-panel">
    <div class="memory-tabs">
      <button class="mem-tab" :class="{ active: memTab === 'summaries' }" @click="memTab = 'summaries'">摘要</button>
      <button class="mem-tab" :class="{ active: memTab === 'files' }" @click="memTab = 'files'">背景</button>
    </div>
    <div class="mem-hint">AI 自动记忆 · 保存章节后自动积累，续写时作为参考</div>

    <div class="mem-body">

      <!-- 章节摘要 -->
      <template v-if="memTab === 'summaries'">
        <div v-if="loadingSummaries" class="mem-loading">加载中...</div>
        <div v-else-if="summaries.length === 0" class="mem-empty">
          还没有摘要<br>
          <span class="hint">保存章节后 AI 会自动生成摘要</span>
        </div>
        <div v-else class="summary-list">
          <div
            v-for="sid in summaries"
            :key="sid"
            class="summary-item"
            :class="{ active: selectedSummary === sid }"
            @click="loadSummary(sid)"
          >
            <span class="summary-icon"><ListTree :size="15"/></span>
            <span class="summary-name">章节 {{ sid }}</span>
          </div>
        </div>
        <div v-if="selectedSummary" class="summary-detail">
          <div class="detail-head">
            <span>章节 {{ selectedSummary }} 摘要</span>
          </div>
          <pre class="char-content">{{ summaryContent }}</pre>
        </div>
      </template>

      <!-- 记忆文件 -->
      <template v-if="memTab === 'files'">
        <div class="file-list">
          <div
            v-for="f in memFiles"
            :key="f.path"
            class="file-item"
            :class="{ active: selectedFile === f.path }"
            @click="loadFile(f.path)"
          >
            <span class="file-icon"><component :is="fileIcon(f.path)" :size="15"/></span>
            <span class="file-name">{{ fileLabel(f.path) }}</span>
          </div>
        </div>
        <div v-if="selectedFile" class="file-detail">
          <div class="detail-head">
            <span>{{ fileLabel(selectedFile) }}</span>
            <button v-if="editingFile" class="btn small primary" @click="saveFile">保存</button>
            <button v-else class="btn small" @click="editingFile = true">编辑</button>
          </div>
          <textarea v-if="editingFile" v-model="fileContent" class="mem-textarea"></textarea>
          <pre v-else class="char-content">{{ fileContent }}</pre>
        </div>
      </template>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, markRaw, type Component } from 'vue'
import { ListTree, Globe2, Calendar, Anchor, PenLine, Settings2, FilePen, Brain, FileText } from 'lucide-vue-next'
import { memoryApi } from '../../core/api'
import { toast } from '../../composables/useToast'

const props = defineProps<{ projectId: string }>()

const memTab = ref<'summaries' | 'files'>('summaries')

// 摘要
const loadingSummaries = ref(false)
const summaries = ref<string[]>([])
const selectedSummary = ref('')
const summaryContent = ref('')

// 记忆文件
const selectedFile = ref('')
const fileContent = ref('')
const editingFile = ref(false)

const memFiles = [
  { path: 'WORLD.md', label: '世界观' },
  { path: 'TIMELINE.md', label: '事件时间线' },
  { path: 'HOOKS.md', label: '伏笔追踪' },
  { path: 'STYLE.md', label: '写作风格' },
  { path: 'PREFERENCES.md', label: '作者偏好' },
  { path: 'working_memory.md', label: '短期记忆' },
  { path: 'long_term.md', label: '长期记忆' },
]

const FILE_ICONS: Record<string, Component> = {
  'WORLD.md': markRaw(Globe2),
  'TIMELINE.md': markRaw(Calendar),
  'HOOKS.md': markRaw(Anchor),
  'STYLE.md': markRaw(PenLine),
  'PREFERENCES.md': markRaw(Settings2),
  'working_memory.md': markRaw(FilePen),
  'long_term.md': markRaw(Brain),
}
function fileIcon(path: string): Component {
  return FILE_ICONS[path] || FileText
}
function fileLabel(path: string): string {
  return memFiles.find(f => f.path === path)?.label || path
}

async function loadSummaries() {
  loadingSummaries.value = true
  try {
    const res = await memoryApi.listSummaries(props.projectId)
    summaries.value = res.data?.data || []
  } catch (e: any) {
    toast.error(e.response?.data?.error || '加载摘要失败')
  } finally {
    loadingSummaries.value = false
  }
}

async function loadSummary(sid: string) {
  selectedSummary.value = sid
  try {
    const res = await memoryApi.getSummary(props.projectId, sid)
    summaryContent.value = res.data?.data?.content || '(空)'
  } catch (e: any) {
    toast.error(e.response?.data?.error || '加载摘要失败')
  }
}

async function loadFile(path: string) {
  selectedFile.value = path
  editingFile.value = false
  try {
    const res = await memoryApi.getFile(props.projectId, path)
    fileContent.value = res.data?.data?.content || '(空)'
  } catch (e: any) {
    fileContent.value = '(空)'
  }
}

async function saveFile() {
  try {
    await memoryApi.writeFile(props.projectId, selectedFile.value, fileContent.value)
    editingFile.value = false
    toast.success('已保存')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '保存失败')
  }
}

watch(memTab, (tab) => {
  if (tab === 'summaries' && summaries.value.length === 0) loadSummaries()
})

// 初始加载
loadSummaries()
</script>

<style scoped>
.memory-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.memory-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  margin: 10px 12px 0;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  flex-shrink: 0;
}
.mem-hint {
  margin: 4px 14px 6px;
  font-size: 10px;
  color: var(--text-tertiary);
  flex-shrink: 0;
}
.mem-tab {
  flex: 1;
  border: none;
  background: transparent;
  padding: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.mem-tab:hover:not(.active) { color: var(--text-primary); }
.mem-tab.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  font-weight: 500;
}

.mem-body {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px 16px;
  min-height: 0;
}

.mem-loading, .mem-empty {
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
  padding: 20px 12px;
  line-height: 1.7;
}
.mem-empty .hint {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  opacity: 0.8;
}

.summary-list, .file-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.summary-item, .file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
  transition: background 0.12s;
}
.summary-item:hover, .file-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
.summary-item.active, .file-item.active {
  background: var(--accent-light);
  color: var(--accent);
}
.summary-icon, .file-icon {
  font-size: 13px;
  flex-shrink: 0;
}
.summary-name, .file-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-detail, .file-detail {
  margin-top: 8px;
  border-top: 1px solid var(--border-subtle, rgba(0,0,0,0.06));
  padding-top: 8px;
}
.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
}
.char-content {
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  margin: 0;
  max-height: 400px;
  overflow-y: auto;
}
.mem-textarea {
  width: 100%;
  min-height: 200px;
  font-size: 12px;
  line-height: 1.6;
  font-family: inherit;
  color: var(--text-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 8px;
  resize: vertical;
  outline: none;
}
.mem-textarea:focus { border-color: var(--accent); }

.btn.small {
  padding: 3px 10px;
  font-size: 11px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
}
.btn.small.primary {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
</style>
