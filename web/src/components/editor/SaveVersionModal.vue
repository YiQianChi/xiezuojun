<template>
  <div v-if="modelValue" class="modal-overlay" @click="cancel">
    <div class="modal save-version-modal" @click.stop>
      <h3>保存版本</h3>
      <p class="modal-tip">请输入保存版本名称，例如 第一版v1.0</p>
      <input
        v-model="label"
        class="input"
        placeholder="版本备注"
        autofocus
        @keyup.enter="confirm"
      />
      <div class="modal-actions">
        <button class="btn" @click="cancel">取消</button>
        <button class="btn primary" @click="confirm" :disabled="saving || !label.trim()">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  save: [label: string]
}>()

const label = ref('第一版v1.0')

// 打开时复位输入值
watch(() => props.modelValue, (v) => {
  if (v) label.value = '第一版v1.0'
})

function cancel() {
  if (props.saving) return
  emit('update:modelValue', false)
}

function confirm() {
  const trimmed = label.value.trim()
  if (!trimmed) return
  emit('save', trimmed)
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
  margin: 0 0 8px 0;
  font-family: var(--font-serif);
  font-size: 18px;
  color: var(--text-primary, #1d1d1f);
}
.modal-tip {
  color: var(--text-secondary, #86868b);
  font-size: 13px;
  margin: 0 0 16px 0;
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
  text-align: center;
}
.input:focus {
  border-color: var(--accent, #007aff);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
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
