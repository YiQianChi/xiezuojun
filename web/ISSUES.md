# 写作菌功能完善清单

> 2026-07-28 基于前后端全量代码审计。后端 100eb76 / 前端 e9ee8e5 + 未提交的右键菜单改动。

## A. 现有功能问题（Bug）

### A1. 审阅跳转基本废了 🔴
- 文件: `Editor.vue:619` `findDocByLocus` + `handleReviewJump`
- 问题: 拿 AI 返回的 locus 字符串模糊匹配文档标题，再拿 desc 前30字去正文 indexOf。AI 返回格式不可控，跳转十有八九跳不对。
- 修复: review mode 的 system prompt 要求 AI 返回 `chapter_title` 字段；前端用 chapter_title 精确匹配文档标题，再用 desc 全文搜索定位段落。如果匹配不到，至少跳到正确章节。

### A2. 审阅报告不持久 🟡
- 文件: `ReviewView.vue`
- 问题: 每次「开始体检」覆盖上次结果，无法对比。
- 修复: 审阅结果存 localStorage（按 project_id），列表显示历史报告时间，可切换查看。

### A3. 没有 Ctrl+S 强制保存 🟡
- 文件: `Editor.vue` 全局 keydown handler
- 问题: 按 Ctrl+S 浏览器弹保存对话框，写作的人本能会按。
- 修复: 拦截 Ctrl+S preventDefault + 触发 autoSave + toast 反馈。

### A4. 没有 Ctrl+F 直接触发搜索 🟡
- 文件: `Editor.vue` 全局 keydown handler
- 问题: 现在要手动点放大镜图标。
- 修复: 拦截 Ctrl+F preventDefault + 打开 SearchPanel。

## B. 缺失功能（按写作者痛感排序）

### B1. 本地草稿备份 🔴
- 文件: `Editor.vue` onUpdate callback
- 问题: autoSave 1.5s debounce 写服务器。浏览器崩了/网断了，1.5s 窗口内的字丢失。
- 修复: onUpdate 时写 `localStorage.setItem('draft-{docId}', text)`，autoSave 成功后 `localStorage.removeItem`。打开文档时检查是否有残留草稿，提示恢复。

### B2. 深色模式 🟡
- 文件: `src/style.css` + 各组件 CSS
- 问题: 写小说的人主力时间是晚上，全亮色刺眼。
- 修复: CSS 变量已全站化，加 `:root.dark` 一套暗色变量 + header toggle 按钮 + localStorage 记住偏好。

### B3. 章节状态标记 🟡
- 文件: `DocTreeNode.vue` + 后端 documents 表
- 问题: 几十章看不清哪些写完没写完。
- 修复: document 加 `status` 字段（draft/revising/final），文档树节点显示彩色圆点标记。右键菜单切换状态。后端 PATCH /documents/:id 已支持扩展字段。

### B4. 快捷键体系 🟢
- 文件: `Editor.vue` handleGlobalKeydown
- 缺: Ctrl+S 保存、Ctrl+F 搜索、Ctrl+P 切模式
- 已有: Ctrl+Shift+Z 切专注

### B5. 分屏参考面板 🟢
- 文件: 新组件 `ReferencePanel.vue`
- 问题: 写作时查设定要切 tab，写作流断了。
- 修复: 编辑器右侧可开一个浮动面板，显示设定文档/角色卡/某章节原文。复用 AIContextPanel 的定位模式。

### B6. 每章目标字数 + 进度 🟢
- 文件: `DocTreeNode.vue` + 后端
- 问题: 只有当前字数，没有目标对比。
- 修复: document 加 `word_goal` 字段，文档树显示进度条。

## C. 部署

### 已有未提交改动
- `Editor.vue` + `editor-page.css`: 右键菜单加了删除线 + 清除高亮
- 需要在本次一起 build + 部署

### 部署流程
- 后端: `go build -o xzj-server-prod ./cmd/server/` → scp → 重启
- 前端: `npm run build` → rsync dist
- 测试服: 10.10.30.180 (systemd=xiezuojun-server)
- 生产: 124.222.138.209 (xiezuojun.com)

## 执行计划

3 个 agent 并行：

1. **Agent A**: 审阅修复（A1 跳转 + A2 持久化）→ ReviewView.vue + Editor.vue(review部分)
2. **Agent B**: 编辑器增强（B1 草稿 + A3/A4 快捷键）→ Editor.vue(save/keyboard部分)
3. **Agent C**: 深色模式（B2）→ style.css + 各CSS

Agent A 和 B 都改 Editor.vue 但改不同区域，patch 精确指定不会冲突。
