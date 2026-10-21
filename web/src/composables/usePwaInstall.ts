import { ref } from 'vue'

// PWA 安装：捕获 beforeinstallprompt，暴露 canInstall + promptInstall
// Windows Chrome/Edge 点「安装」后会在桌面/开始菜单生成「写作菌」图标，双击开独立窗口

type BIPEvent = Event & {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

let deferredPrompt: BIPEvent | null = null
const canInstall = ref(false)
const isStandalone = ref(false)

if (typeof window !== 'undefined') {
  isStandalone.value =
    window.matchMedia('(display-mode: standalone)').matches ||
    (window.navigator as unknown as { standalone?: boolean }).standalone === true

  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault()
    deferredPrompt = e as BIPEvent
    canInstall.value = true
  })

  window.addEventListener('appinstalled', () => {
    deferredPrompt = null
    canInstall.value = false
  })

  // 注册 service worker（仅生产，https/localhost 才可用）
  if ('serviceWorker' in navigator && location.protocol === 'https:') {
    window.addEventListener('load', () => {
      navigator.serviceWorker.register('/sw.js').catch(() => {})
    })
  }
}

async function promptInstall(): Promise<'accepted' | 'dismissed' | 'unavailable'> {
  if (!deferredPrompt) return 'unavailable'
  await deferredPrompt.prompt()
  const { outcome } = await deferredPrompt.userChoice
  deferredPrompt = null
  canInstall.value = false
  return outcome
}

export function usePwaInstall() {
  return { canInstall, isStandalone, promptInstall }
}
