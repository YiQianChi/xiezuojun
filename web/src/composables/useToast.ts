import { ref } from 'vue'

export interface ToastItem {
  id: number
  type: 'error' | 'success' | 'info'
  message: string
}

export const toasts = ref<ToastItem[]>([])
let nextId = 0

function show(type: ToastItem['type'], message: string, duration = 3500) {
  const id = ++nextId
  toasts.value.push({ id, type, message })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, duration)
}

export function useToast() {
  return {
    toasts,
    error: (msg: string) => show('error', msg),
    success: (msg: string) => show('success', msg, 2500),
    info: (msg: string) => show('info', msg),
  }
}

// 全局快捷方法，供非组件代码（composables 等）使用
export const toast = {
  error: (msg: string) => show('error', msg),
  success: (msg: string) => show('success', msg, 2500),
  info: (msg: string) => show('info', msg),
}
