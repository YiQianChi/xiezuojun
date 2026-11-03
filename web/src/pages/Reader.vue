<template>
  <div class="reader">
    <header class="rd-head">
      <img src="/icons/icon-192.png" alt="写作菌" class="rd-logo" />
      <span class="rd-brand">写作菌</span>
      <router-link to="/" class="rd-cta">去创作 →</router-link>
    </header>

    <main v-if="chapter" class="rd-main">
      <p class="rd-book">{{ chapter.project_name }}</p>
      <h1 class="rd-title">{{ chapter.title }}</h1>
      <p class="rd-meta">{{ chapter.word_count }} 字 · 更新于 {{ chapter.updated_at }}</p>
      <!-- 作者自己的富文本内容；服务端来源可信，仅做基础净化 -->
      <div class="rd-content" v-html="safeContent"></div>
      <div class="rd-end">— 完 —</div>
      <div class="rd-card">
        <p>也想安安静静写小说？</p>
        <router-link to="/" class="btn primary">免费使用写作菌</router-link>
      </div>
    </main>

    <div v-else-if="!loading" class="rd-none">
      <p>分享不存在或已被作者关闭</p>
      <router-link to="/" class="btn">去写作菌看看</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { documentApi } from '../core/api'

const route = useRoute()
const chapter = ref<any>(null)
const loading = ref(true)

// 基础净化：去 script/iframe/on* 属性（来源是作者自己保存的正文，非任意第三方）
const safeContent = computed(() => {
  let s = chapter.value?.content || '（本章还没有内容）'
  s = s.replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<iframe[\s\S]*?<\/iframe>/gi, '')
    .replace(/\son\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi, '')
    .replace(/javascript:/gi, '')
  return s
})

onMounted(async () => {
  try {
    const res = await documentApi.publicRead(String(route.params.code))
    chapter.value = res.data?.data || res.data
  } catch { /* 404 落空态 */ }
  loading.value = false
  document.title = chapter.value ? `${chapter.value.title} · 写作菌` : '写作菌'
})
</script>

<style scoped>
.reader { max-width: 680px; margin: 0 auto; padding: 0 20px 60px; }

.rd-head { display: flex; align-items: center; gap: 8px; padding: 14px 0; border-bottom: 1px solid var(--border); }
.rd-logo { width: 26px; height: 26px; border-radius: 7px; }
.rd-brand { font-weight: 700; font-size: 16px; color: var(--text-primary); }
.rd-cta { margin-left: auto; font-size: 13px; color: var(--accent); text-decoration: none; }

.rd-main { padding-top: 28px; }
.rd-book { font-size: 13px; color: var(--text-secondary); }
.rd-title { font-size: 24px; font-weight: 800; color: var(--text-primary); margin: 6px 0 4px; }
.rd-meta { font-size: 12px; color: var(--text-tertiary); margin-bottom: 24px; }

.rd-content { font-size: 17px; line-height: 1.9; color: var(--text-primary); word-break: break-word; }
.rd-content :deep(p) { margin: 0 0 1.1em; }
.rd-content :deep(h1), .rd-content :deep(h2), .rd-content :deep(h3) { font-size: 1.15em; margin: 1.4em 0 0.7em; }
.rd-end { text-align: center; color: var(--text-tertiary); font-size: 13px; margin: 36px 0; }

.rd-card { text-align: center; background: var(--bg-secondary); border-radius: 16px; padding: 22px 16px; }
.rd-card p { font-size: 14px; color: var(--text-secondary); margin-bottom: 12px; }

.rd-none { text-align: center; padding: 80px 0; color: var(--text-secondary); font-size: 14px; }
.rd-none .btn { margin-top: 14px; }
</style>
