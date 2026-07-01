<template>
  <div class="home">
    <div class="home-hero">
      <h1 class="hero-title">写作菌</h1>
      <p class="hero-tagline">专注小说创作的 AI 写作工具</p>
      <p class="hero-desc">让灵感不被打扰，让创作更加纯粹</p>
      <div class="home-actions">
        <router-link to="/bookshelf" class="btn primary big">进入书架</router-link>
        <router-link v-if="!isLoggedIn" to="/register" class="btn big">注册</router-link>
      </div>
    </div>

    <div class="features">
      <div class="feature">
        <div class="feature-icon">
          <!-- 钢笔尖 · 写作 -->
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14.4 3.9 20.1 9.6"/>
            <path d="M17.3 7 4.3 20 3 21l1-1.3L17 6.7"/>
            <path d="m15 5 4 4"/>
            <path d="M6 18l-1.5-1.5"/>
          </svg>
        </div>
        <h3>纯粹写作</h3>
        <p>沉浸式编辑器，干净的页面，让你专注于文字</p>
      </div>
      <div class="feature">
        <div class="feature-icon">
          <!-- 闪光星 · AI -->
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 3v3"/>
            <path d="M12 18v3"/>
            <path d="M3 12h3"/>
            <path d="M18 12h3"/>
            <path d="M12 8.5c0 1.9 1.6 3.5 3.5 3.5-1.9 0-3.5 1.6-3.5 3.5 0-1.9-1.6-3.5-3.5-3.5 1.9 0 3.5-1.6 3.5-3.5Z" fill="currentColor" fill-opacity="0.15"/>
            <path d="m18.5 4.5.5 1.5 1.5.5-1.5.5-.5 1.5-.5-1.5-1.5-.5 1.5-.5.5-1.5Z"/>
            <path d="m5 16 .4 1.2L6.6 17.6l-1.2.4L5 19.2l-.4-1.2L3.4 17.6l1.2-.4L5 16Z"/>
          </svg>
        </div>
        <h3>AI 助手</h3>
        <p>续写、改写、润色、扩写，AI 与你一起创作</p>
      </div>
      <div class="feature">
        <div class="feature-icon">
          <!-- 书本 · 章节 -->
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 4.5A2 2 0 0 1 6 2.5h13v16H6a2 2 0 0 0-2 2V4.5Z"/>
            <path d="M4 20.5A2 2 0 0 1 6 18.5h13"/>
            <path d="M8 7h7"/>
            <path d="M8 11h5"/>
          </svg>
        </div>
        <h3>章节管理</h3>
        <p>大纲、人物、世界观，结构清晰</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
const isLoggedIn = ref(!!localStorage.getItem('access_token'))
</script>

<style scoped>
.home {
  max-width: 1040px;
  margin: 0 auto;
  padding: 40px 32px 56px;
}

.home-hero {
  text-align: center;
  padding: 32px 0 44px;
}

.hero-title {
  font-family: var(--font-serif);
  font-size: 56px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 14px;
  letter-spacing: 4px;
}

.hero-tagline {
  font-size: 19px;
  color: var(--text-primary);
  margin-bottom: 8px;
  font-weight: 500;
}

.hero-desc {
  font-size: 15px;
  color: var(--text-secondary);
  margin-bottom: 30px;
}

.home-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.btn.big {
  padding: 10px 26px;
  font-size: 14px;
  border-radius: var(--radius-md);
}

.features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-top: 8px;
}

.feature {
  background: var(--bg-secondary);
  padding: 24px 22px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle, rgba(0, 0, 0, 0.04));
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  text-align: center;
  transition: all var(--transition-normal);
}

.feature:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 122, 255, 0.08), 0 1px 3px rgba(0, 0, 0, 0.04);
  border-color: rgba(0, 122, 255, 0.15);
}

.feature-icon {
  width: 44px;
  height: 44px;
  margin: 0 auto 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(0, 122, 255, 0.10) 0%, rgba(0, 122, 255, 0.04) 100%);
  color: var(--accent, #007aff);
}

.feature-icon svg {
  width: 22px;
  height: 22px;
}

.feature h3 {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--text-primary);
  letter-spacing: 0.2px;
}

.feature p {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

@media (max-width: 720px) {
  .hero-title { font-size: 44px; }
  .features { grid-template-columns: 1fr; }
  .home-hero { padding: 20px 0 32px; }
}
</style>
