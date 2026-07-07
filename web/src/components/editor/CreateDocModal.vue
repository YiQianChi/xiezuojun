<template>
  <div v-if="modelValue" class="modal-overlay" @click="close">
    <div class="modal" @click.stop>
      <h3>{{ isFolder ? '新建文件夹' : (props.sideTab === 'setting' ? '新建设定' : '新建章节') }}</h3>
      <input
        v-model="name"
        :placeholder="isFolder ? '文件夹名' : (props.sideTab === 'setting' ? '设定名' : '章节名')"
        class="input"
        @keyup.enter="confirm"
        autofocus
      />
      <div v-if="isFolder && folderTypeOptions?.length" class="folder-type-row">
        <label class="folder-type-label">类型</label>
        <div class="folder-type-options">
          <button
            v-for="opt in folderTypeOptions"
            :key="opt.value"
            class="folder-type-chip"
            :class="{ active: selectedFolderType === opt.value }"
            @click="selectedFolderType = opt.value"
          >{{ opt.label }}</button>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn" @click="close">取消</button>
        <button class="btn primary" @click="confirm" :disabled="!name.trim()">创建</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  docType: string
  sideTab?: 'chapter' | 'setting'
  folderTypeOptions?: { value: string; label: string }[]
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  create: [name: string, folderType?: string]
}>()

const name = ref('')
const selectedFolderType = ref('')
const isFolder = computed(() => {
  const t = props.docType
  return t === 'volume' || ['outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline'].includes(t)
})

watch(() => props.folderTypeOptions, (opts) => {
  if (opts && opts.length) selectedFolderType.value = opts[0].value
}, { immediate: true })

// 打开时清空
watch(() => props.modelValue, (v) => {
  if (v) {
    name.value = ''
    if (props.folderTypeOptions?.length) selectedFolderType.value = props.folderTypeOptions[0].value
  }
})

function close() {
  emit('update:modelValue', false)
}

function confirm() {
  const trimmed = name.value.trim()
  if (!trimmed) return
  emit('create', trimmed, selectedFolderType.value || undefined)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 500;
}

.modal {
  width: 400px;
  max-width: 90vw;
  background: var(--bg-secondary, #fff);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  padding: 24px;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}

h3 {
  margin: 0 0 16px 0;
  font-family: var(--font-serif);
  font-size: 18px;
  color: var(--text-primary, #1d1d1f);
}

.input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  outline: none;
  font-size: 14px;
  color: var(--text-primary, #1d1d1f);
  box-sizing: border-box;
}
.input:focus {
  border-color: var(--accent, #007aff);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.folder-type-row {
  margin-top: 14px;
}
.folder-type-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary, #86868b);
  margin-bottom: 6px;
}
.folder-type-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.folder-type-chip {
  padding: 4px 10px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #fff);
  font-size: 12px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
  transition: all 0.15s;
}
.folder-type-chip.active {
  border-color: var(--accent, #007aff);
  background: rgba(0, 122, 255, 0.08);
  color: var(--accent, #007aff);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}
.btn {
  padding: 6px 14px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #fff);
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
}
.btn.primary {
  background: var(--accent, #007aff);
  color: white;
  border-color: var(--accent, #007aff);
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
