<template>
  <div class="editor-header">
    <div class="header-left">
      <button class="btn icon-btn menu-btn" @click="$emit('toggle-sidebar')" title="目录">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M3 6h18M3 12h18M3 18h18"/></svg>
      </button>
      <button @click="$emit('back')" class="back-btn" title="返回书架">←</button>
    </div>
    <div class="doc-name">
      <span class="proj-name">{{ projectName || '未命名项目' }}</span>
      <template v-if="currentDocName && mode === 'write'">
        <span class="doc-sep">·</span>
        <span class="chapter-name">{{ currentDocName }}</span>
      </template>
    </div>

    <!-- 构思模式已删（2026-11-01 Boss 指令）；只剩写作+隐藏的审核，单一 tab 无意义整个切换器暂藏——恢复审核时去掉 v-if -->
    <div class="mode-switcher" role="tablist" v-if="false">
      <button
        class="mode-tab write"
        :class="{ active: mode === 'write' }"
        @click="$emit('switch-mode', 'write')"
        title="写作：正文创作 + AI 续写/改写"
      ><span class="dot"></span>写作</button>
      <!-- 审核功能先隐藏（2026-10-22 Boss 指示），恢复时删掉 v-if 即可 -->
      <button
        v-if="false"
        class="mode-tab review"
        :class="{ active: mode === 'review' }"
        @click="$emit('switch-mode', 'review')"
        title="审核：人设一致性、情节漏洞、伏笔、节奏"
      ><span class="dot"></span>审核</button>
    </div>

    <div class="header-actions">
      <!-- 方案一·瘦身：主题/分屏参考/版本类低频功能全收进 ⋯ 菜单，顶栏只留 模式+专注+⋯ -->
      <div class="theme-picker" v-if="themeMenuOpen" @click.stop />
      <div class="theme-pop" v-if="themeMenuOpen">
        <div class="tp-title">皮肤</div>
        <button
          v-for="t in THEMES" :key="t.id"
          class="tp-item" :class="{ on: themeId === t.id }"
          @click="pickTheme(t.id)"
        >
          <span class="tp-sw" :style="{ background: t.sw }"><i :style="{ background: t.sw2 }"></i></span>
          <span class="tp-name">{{ t.name }}</span>
          <span class="tp-sub">{{ t.desc }}</span>
        </button>
      </div>
      <div class="more-wrap" @click.stop>
        <button
          @click.stop="moreMenuOpen = !moreMenuOpen"
          class="btn icon-btn"
          :class="{ active: moreMenuOpen || themeMenuOpen }"
          title="更多"
        >⋯</button>
        <div class="more-pop" v-if="moreMenuOpen">
          <button class="mp-item" @click="moreMenuOpen = false; themeMenuOpen = true">🎨 主题（皮肤）</button>
          <button v-if="mode === 'write'" class="mp-item" @click="moreMenuOpen = false; $emit('toggle-reference')">📖 分屏</button>
          <button v-if="mode === 'write'" class="mp-item" :disabled="!activeDocId || savingVersion" @click="moreMenuOpen = false; $emit('save-version')">💾 保存版本</button>
          <button v-if="mode === 'write'" class="mp-item" :disabled="!activeDocId" @click="moreMenuOpen = false; $emit('open-history')">🕒 版本历史</button>
        </div>
      </div>
      <button
        v-if="mode === 'write'"
        @click="$emit('toggle-focus')"
        class="btn icon-btn focus-btn"
        :class="{ active: focusMode }"
        :title="focusMode ? '退出专注模式 (Esc)' : '专注模式 (Ctrl+Shift+Z)'"
      >
        <svg v-if="!focusMode" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M16 3h3a2 2 0 0 1 2 2v3"/><path d="M8 21H5a2 2 0 0 1-2-2v-3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>
        <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

type EditorMode = 'write' | 'review' // 构思已删（2026-11-01）；审核隐藏保留

defineProps<{
  projectName: string
  currentDocName: string
  mode: EditorMode
  activeDocId: number | null
  savingVersion: boolean
  focusMode: boolean
  referenceOpen: boolean
}>()

defineEmits<{
  back: []
  'switch-mode': [mode: EditorMode]
  'save-version': []
  'open-history': []
  'toggle-focus': []
  'toggle-reference': []
  'toggle-sidebar': []
}>()

const THEMES = [
  { id: 'xuan', name: '米宣', desc: '默认 · 暖宣纸', sw: '#f8f3e8', sw2: '#fbf7ee' },
  { id: 'ios', name: 'iOS', desc: '苹果灰 · 简洁', sw: '#f2f2f7', sw2: '#ffffff' },
  { id: 'qing', name: '青竹', desc: '淡青绿 · 护眼', sw: '#e4eee4', sw2: '#f4faf4' },
  { id: 'wu', name: '晨雾', desc: '冷灰蓝 · 清爽', sw: '#e5eaf1', sw2: '#f5f8fb' },
  { id: 'yangpi', name: '仿古', desc: '羊皮纸 · 怀旧', sw: '#e9dec4', sw2: '#f7efdb' },
  { id: 'ye', name: '夜灯', desc: '暖墨绿 · 深夜', sw: '#2b2e26', sw2: '#2f322a' },
  { id: 'mo', name: '冷墨', desc: '标准深色', sw: '#2c2c2e', sw2: '#1a1a1c' },
]
const themeId = ref(localStorage.getItem('xzj-theme') || (localStorage.getItem('theme') === 'dark' ? 'mo' : 'xuan'))
const themeMenuOpen = ref(false)
const moreMenuOpen = ref(false)

function applyThemeClass(t: string) {
  const el = document.documentElement
  el.classList.remove('dark', 'theme-ye', 'theme-qing', 'theme-wu', 'theme-yangpi', 'theme-ios')
  if (t === 'mo') el.classList.add('dark')
  else if (t === 'ye') el.classList.add('dark', 'theme-ye')
  else if (['qing', 'wu', 'yangpi', 'ios'].includes(t)) el.classList.add('theme-' + t)
}

function pickTheme(t: string) {
  themeId.value = t
  applyThemeClass(t)
  localStorage.setItem('xzj-theme', t)
  localStorage.setItem('theme', (t === 'mo' || t === 'ye') ? 'dark' : 'light') // 旧键兼容其它页面
  themeMenuOpen.value = false
}

// 点外部收起（主题弹层 + ⋯ 菜单）
function onDocClickCloseTheme() { themeMenuOpen.value = false; moreMenuOpen.value = false }
if (typeof document !== 'undefined') {
  document.addEventListener('click', onDocClickCloseTheme)
}
</script>

<style scoped>
.header-left {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.menu-btn { display: none; }

.editor-header {
  height: 52px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  /* backdrop-filter 会创建层叠上下文，把弹层（主题选择）困在 header 层；
     提到右栏(300)/搭子窗(900) 之间——主题弹层要压右栏 */
  position: relative;
  z-index: 950;
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

/* ⋯ 更多菜单：低频功能（主题/分屏参考/版本）收纳 */
.more-wrap { position: relative; }
.more-pop {
  position: absolute;
  top: 38px;
  right: 0;
  z-index: 660;
  background: var(--ed-panel, #f8f3e8);
  border: 1px solid var(--ed-line, #e2d8c2);
  border-radius: 12px;
  box-shadow: 0 8px 28px var(--ed-shadow, rgba(96, 74, 40, 0.16));
  padding: 6px;
  min-width: 150px;
}
.mp-item {
  display: block;
  width: 100%;
  border: none;
  background: transparent;
  border-radius: 8px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--ed-ink, #33302a);
  text-align: left;
  cursor: pointer;
}
.mp-item:hover { background: var(--ed-hover, rgba(150, 130, 100, 0.09)); }
.mp-item:disabled { opacity: 0.45; cursor: not-allowed; }

/* 专注=顶栏唯一主色按钮 */
.btn.icon-btn.focus-btn:not(.active) {
  background: linear-gradient(135deg, #8fc59b, #6aa877);
  color: #fff;
  border-color: transparent;
}

/* ponytail-mobile: 窄屏——汉堡出、返回文字缩、模式切换和操作按钮收紧 */
@media (max-width: 768px) {
  .editor-header { padding: 0 10px; gap: 8px; height: 48px; }
  .menu-btn { display: inline-flex; }
  .back-btn { font-size: 12px; padding: 6px 4px; }
  .doc-name { font-size: 13px; min-width: 0; flex: 1; }
  .proj-name { max-width: 30%; font-size: 12px; }
  .doc-name .doc-sep { display: none; }
  .mode-switcher { margin: 0 4px; }
  .mode-tab { padding: 5px 8px; font-size: 12px; }
  .mode-tab .dot { display: none; }
  .mode-tab.review { display: none; }
  .header-actions .btn:not(.icon-btn) { display: none; }
  .header-actions { gap: 4px; }
}

/* 主题选择弹层：贴在调色盘按钮下方（header-actions 尾部，向左展开） */
.theme-pop {
  position: absolute;
  top: 40px;
  right: 10px;
  z-index: 660;
  background: var(--ed-panel, #f8f3e8);
  border: 1px solid var(--ed-line, #e2d8c2);
  border-radius: 12px;
  box-shadow: 0 8px 28px var(--ed-shadow, rgba(96, 74, 40, 0.16));
  padding: 6px;
  min-width: 190px;
}
.tp-title {
  font-size: 10px;
  color: var(--ed-ink-soft, #a39a86);
  padding: 4px 8px 6px;
  letter-spacing: 2px;
}
.tp-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  border: none;
  background: transparent;
  border-radius: 8px;
  padding: 7px 8px;
  cursor: pointer;
  color: var(--ed-ink, #33302a);
  text-align: left;
}
.tp-item:hover { background: var(--ed-hover, rgba(150, 130, 100, 0.09)); }
.tp-item.on { background: var(--ed-hover, rgba(150, 130, 100, 0.09)); }
.tp-item.on .tp-name { color: #6aa877; font-weight: 700; }
.tp-sw {
  width: 22px; height: 22px;
  border-radius: 6px;
  border: 1px solid var(--ed-line, #e2d8c2);
  display: flex; align-items: flex-end; justify-content: flex-end;
  padding: 2px;
  flex-shrink: 0;
}
.tp-sw i { width: 10px; height: 10px; border-radius: 3px; display: block; }
.tp-name { font-size: 12.5px; }
.tp-sub { font-size: 10px; color: var(--ed-ink-soft, #a39a86); margin-left: auto; }
</style>
