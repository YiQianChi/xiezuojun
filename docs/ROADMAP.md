# 写作菌路线图与迭代记录

> 合并自原 web/TODO.md（2026-07-02 审计计划）与 web/ISSUES.md（2026-07-28 功能清单）。
> 完成项保留作记录，未完成项是当前待办。

## ✅ 已完成

### 第 1 轮：Bug 修复 + 基础设施（2026-07-02 计划）

- ✅ 异步记忆提取 context 取消 → `context.Background()`
- ✅ 原子写入加 fsync，断电不丢数据
- ✅ 用量记录 UserID 修正（原误用 ProjectID）
- ✅ Token 自动刷新（401 → refresh → 重试）
- ✅ Toast 组件替换全站 alert()
- ✅ 记忆提取模型名改环境变量（原硬编码 glm-5-turbo 对火山 ark 404）

### 第 2 轮：记忆系统 UI

- ✅ 角色卡面板 / 章节摘要 / 记忆文件编辑（memoryApi 前端对接）

### 第 3 轮：回收站 + 用户中心

- ✅ 回收站（列表/恢复/清空）
- ✅ 用户中心、项目重命名、项目导出、密码重置
- ✅ 配额 tier 从 user.Plan 读取

### P0 三件套（2026-07-28）

- ✅ C1. 写作统计面板（今日/本周/本月/连续天数/14天柱状图）
- ✅ C2. 章节大纲卡片视图（StoryboardView，拖拽重排）
- ✅ C3. 审阅问题标记已解决/忽略 + 过滤栏

### P1/P2（2026-07-28）

- ✅ D1. DOCX 导出（标准库 ZIP+XML）
- ✅ D2. AI 对话（SSE 流式多轮）
- ✅ D3. 记忆面板扩容
- ✅ D4. 写作计时器
- ✅ D5. 参考面板保留格式
- ✅ E1. 场景分隔/导航（Alt+↑↓）
- ✅ E2. TTS 语音选择
- ✅ E5. 快捷键面板（Ctrl+/）
- ✅ 编辑器新手引导（欢迎卡 + spotlight + AI 演示 overlay）
- ✅ SEO：SPA 壳 canonical/og/JSON-LD、sitemap、爬虫专用静态首页

## ⬜ 待做

- E3. 角色出场追踪 — 点角色名看出场章节统计
- E4. 内联 AI 问答 — 写到一半侧边栏直接问 AI
- 关系图谱 — 方案见 [design/relationship-graph-plan.md](design/relationship-graph-plan.md)
- GET /projects/search、GET /ai/models 前端对接（低优）
- 存储层重设计 — 方案见 [design/STORAGE_REDESIGN.md](design/STORAGE_REDESIGN.md)
- （182 遗留）TTS 相关半成品改动未提交：server 的 main.go 修改 + internal/tts/handler.go 删除，接手时先评估
