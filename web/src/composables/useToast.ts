import { ref } from 'vue'

export interface ToastAction {
  label: string
  onClick: () => void
}

export interface ToastItem {
  id: number
  type: 'error' | 'success' | 'info'
  message: string
  action?: ToastAction
}

export const toasts = ref<ToastItem[]>([])
let nextId = 0

function show(type: ToastItem['type'], message: string, duration = 3500, action?: ToastAction) {
  const id = ++nextId
  toasts.value.push({ id, type, message, action })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, action ? Math.max(duration, 6000) : duration) // 带操作按钮的至少留 6 秒，给误操作留撤回窗口
}

export function dismissToast(id: number) {
  toasts.value = toasts.value.filter(t => t.id !== id)
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
  // 带操作按钮的提示（如「撤回」）——误操作可即时挽回
  withAction: (msg: string, action: ToastAction, duration = 6000) => show('info', msg, duration, action),
}
