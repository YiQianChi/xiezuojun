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
        <div class="feature-icon">✍️</div>
        <h3>纯粹写作</h3>
        <p>沉浸式编辑器，干净的页面，让你专注于文字</p>
      </div>
      <div class="feature">
        <div class="feature-icon">🤖</div>
        <h3>AI 助手</h3>
        <p>续写、改写、润色、扩写，AI 与你一起创作</p>
      </div>
      <div class="feature">
        <div class="feature-icon">📚</div>
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
  max-width: 1100px;
  margin: 0 auto;
  padding: 60px 32px;
}

.home-hero {
  text-align: center;
  padding: 60px 0 80px;
}

.hero-title {
  font-family: var(--font-serif);
  font-size: 64px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 20px;
  letter-spacing: 4px;
}

.hero-tagline {
  font-size: 22px;
  color: var(--text-primary);
  margin-bottom: 12px;
  font-weight: 500;
}

.hero-desc {
  font-size: 16px;
  color: var(--text-secondary);
  margin-bottom: 48px;
}

.home-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.btn.big {
  padding: 12px 32px;
  font-size: 15px;
  border-radius: var(--radius-md);
}

.features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  margin-top: 40px;
}

.feature {
  background: var(--bg-secondary);
  padding: 36px 28px;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  text-align: center;
  transition: all var(--transition-normal);
}

.feature:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
}

.feature-icon {
  font-size: 36px;
  margin-bottom: 16px;
}

.feature h3 {
  font-size: 17px;
  font-weight: 600;
  margin-bottom: 10px;
  color: var(--text-primary);
}

.feature p {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

@media (max-width: 720px) {
  .hero-title { font-size: 48px; }
  .features { grid-template-columns: 1fr; }
}
</style>