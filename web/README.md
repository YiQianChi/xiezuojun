# 写作菌前端（web/）

Vue 3 + TypeScript + Vite + Tiptap 3 的小说创作编辑器。属于 [xiezuojun](../README.md) monorepo 的一部分，后端见 [server/](../server/README.md)。

## 技术栈

- Vue 3 `<script setup>` + TypeScript + Vite
- Tiptap 3 富文本编辑器
- SSE 流式 AI 对话/续写

## 开发

```bash
npm install
npm run dev      # 本地开发
npm run build    # 产出 dist/
```

后端地址等环境配置见 `src/core/api.ts`。

## 目录速览

```
src/
  components/    — 编辑器面板（EditorSidebar/EditorHeader/AIContextPanel/StoryboardView…）
  composables/   — useToast / useAIToolbar / useVersionManager 等逻辑复用
  core/          — api.ts 后端对接层
  pages/         — 书架 Bookshelf、编辑器 Editor 等
```

完整组件地图：[../docs/PROJECT_MAP.md](../docs/PROJECT_MAP.md)

## 功能与快捷键

- Ctrl+P 循环 Plan→Write→Review 模式；Ctrl+/ 快捷键面板；Ctrl+S 强制保存；Ctrl+F 搜索
- Alt+↑↓ 场景跳转；章节状态彩色圆点；分屏参考面板；写作计时器/统计面板
- 更多见 [../docs/ROADMAP.md](../docs/ROADMAP.md)

## 部署

`npm run build` 后 rsync `dist/` 到服务器 `/opt/xiezuojun-web/dist/`（测试服 180 先行验证）。详见 [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)。
