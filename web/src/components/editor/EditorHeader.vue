<template>
  <div class="editor-header">
    <button @click="$emit('back')" class="back-btn">← 返回书架</button>
    <div class="doc-name">
      <span class="proj-name">{{ projectName || '未命名项目' }}</span>
      <template v-if="currentDocName && mode === 'write'">
        <span class="doc-sep">·</span>
        <span class="chapter-name">{{ currentDocName }}</span>
      </template>
    </div>

    <div class="mode-switcher" role="tablist">
      <button
        class="mode-tab plan"
        :class="{ active: mode === 'plan' }"
        @click="$emit('switch-mode', 'plan')"
        title="构思：梗概/世界观/人物/大纲"
      ><span class="dot"></span>构思</button>
      <button
        class="mode-tab write"
        :class="{ active: mode === 'write' }"
        @click="$emit('switch-mode', 'write')"
        title="写作：正文创作 + AI 续写/改写"
      ><span class="dot"></span>写作</button>
      <button
        class="mode-tab review"
        :class="{ active: mode === 'review' }"
        @click="$emit('switch-mode', 'review')"
        title="审核：人设一致性、情节漏洞、伏笔、节奏"
      ><span class="dot"></span>审核</button>
    </div>

    <div class="header-actions">
      <button
        @click="toggleDark"
        class="btn icon-btn"
        :title="isDark ? '切换浅色模式' : '切换深色模式'"
      >
        <svg v-if="!isDark" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
        <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
      </button>
      <button
        v-if="mode === 'write'"
        @click="$emit('toggle-focus')"
        class="btn icon-btn"
        :class="{ active: focusMode }"
        :title="focusMode ? '退出专注模式 (Esc)' : '专注模式 (Ctrl+Shift+Z)'"
      >
        <svg v-if="!focusMode" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M16 3h3a2 2 0 0 1 2 2v3"/><path d="M8 21H5a2 2 0 0 1-2-2v-3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>
        <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
      </button>
      <button
        v-if="mode === 'write'"
        @click="$emit('save-version')"
        class="btn"
        :disabled="!activeDocId || savingVersion"
      >
        {{ savingVersion ? '保存中...' : '保存版本' }}
      </button>
      <button
        v-if="mode === 'write'"
        @click="$emit('open-history')"
        class="btn"
        :disabled="!activeDocId"
      >版本历史</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

type EditorMode = 'plan' | 'write' | 'review'

defineProps<{
  projectName: string
  currentDocName: string
  mode: EditorMode
  activeDocId: number | null
  savingVersion: boolean
  focusMode: boolean
}>()

defineEmits<{
  back: []
  'switch-mode': [mode: EditorMode]
  'save-version': []
  'open-history': []
  'toggle-focus': []
}>()

const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleDark() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>

<style scoped>
.editor-header {
  height: 52px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 16px;
  border-bottom: 1px solid var(--border, #e5e5ea);
  background: var(--bg-secondary, #ffffff);
  flex-shrink: 0;
}

.back-btn {
  border: none;
  background: transparent;
  color: var(--accent, #007aff);
  font-size: 14px;
  cursor: pointer;
  padding: 6px 8px;
}

.doc-name {
  flex: 1;
  min-width: 0;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: baseline;
  gap: 8px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.proj-name {
  color: var(--text-primary);
  font-weight: 600;
  flex-shrink: 0;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
}
.doc-sep {
  color: var(--text-tertiary, #aeaeb2);
  font-weight: 400;
  flex-shrink: 0;
}
.chapter-name {
  color: var(--text-secondary, #86868b);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

/* iOS 分段控件 */
.mode-switcher {
  display: inline-flex;
  background: var(--segment-bg);
  border-radius: 10px;
  padding: 3px;
  gap: 2px;
  margin: 0 12px;
}
.mode-tab {
  border: none;
  background: transparent;
  padding: 6px 16px;
  font-size: 13px;
  color: var(--text-secondary);
  border-radius: 7px;
  cursor: pointer;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s ease;
}
.mode-tab .dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: currentColor;
  opacity: 0.7;
}
.mode-tab:hover:not(.active) { color: var(--text-primary); }
.mode-tab.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: 0 1px 3px var(--shadow-color);
  font-weight: 600;
}
.mode-tab.plan.active { color: #af52de; }
.mode-tab.write.active { color: #007aff; }
.mode-tab.review.active { color: #ff9500; }

.header-actions {
  display: flex;
  gap: 8px;
}
.btn {
  padding: 6px 14px;
  border: 1px solid var(--border, #e5e5ea);
  border-radius: 8px;
  background: var(--bg-secondary, #ffffff);
  font-size: 13px;
  color: var(--text-primary, #1d1d1f);
  cursor: pointer;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
}
.btn.icon-btn.active {
  background: var(--accent, #007aff);
  color: #fff;
  border-color: var(--accent, #007aff);
}
</style>
