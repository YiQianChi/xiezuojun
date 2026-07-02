# 写作菌完善计划

> 生成于 2026-07-02，基于前后端代码审计。后端 83a9397 / 前端 8b2157c。

## 审计结论

后端 50+ API，前端只对接了 15 个。26 个后端 API 前端零调用。
3 个 bug 导致已有功能静默失效。记忆系统后端完整但前端完全缺失。

---

## 第 1 轮：Bug 修复 + 前端基础设施

### Bug 1: 异步记忆提取 context 被取消
- 文件: `cmd/server/main.go:214`
- 问题: `c.Request.Context()` 在 HTTP 响应后取消，AI 记忆提取静默失败
- 修复: 改 `context.Background()`

### Bug 2: 原子写入没有 fsync
- 文件: `internal/store/atomic.go:24`
- 问题: 写 tmp 文件后直接 rename，没 fsync，断电丢数据
- 修复: rename 前 `f.Sync()`

### Bug 3: 用量记录 UserID 写错
- 文件: `internal/gateway/gateway.go:130`
- 问题: `UserID: req.ProjectID`，把项目 ID 当用户 ID
- 修复: 从请求传入真实 userID（需在 HandleGenerate 提取 JWT userID）

### Token 自动刷新
- 文件: `src/core/api.ts`
- 问题: access token 2h 过期直接踢登录页，refresh_token 存了没用
- 修复: axios 响应拦截器 401 → POST /auth/refresh → 重试原请求 → 失败才跳登录

### Toast 通知组件
- 新文件: `src/composables/useToast.ts`
- 问题: 全站用 `alert()` 报错
- 修复: 最轻 toast composable + 固定定位 div，替换 alert()
- 影响文件: useAIToolbar.ts, useVersionManager.ts, Bookshelf.vue, Editor.vue 等 ~8 处 alert

---

## 第 2 轮：记忆系统 UI

后端 12 个 `/memory/*` API 全就绪，前端零 UI。

### 角色卡面板
- 对接: `GET/PUT/DELETE /memory/characters`, `GET /memory/characters/:name`
- 位置: EditorSidebar 新增 tab 或 AIContextPanel 扩展
- 功能: 列出角色卡、查看/编辑/删除。AI 自动提取的角色在这里可见可改

### 章节摘要
- 对接: `GET /memory/summaries`, `GET /memory/summaries/:chapter_id`
- 位置: 文档树章节节点上显示摘要标记，点击查看

### 记忆文件编辑
- 对接: `GET/PUT /memory/file`, `GET /memory/:project_id`
- 位置: 设置面板加「世界观/时间线/伏笔」可编辑区域
- 已有: AIContextPanel.vue 有 context chip 展示逻辑，扩展它

### 前端需新增的 API 调用 (src/core/api.ts)
```
memoryApi = {
  listCharacters(projectId), getCharacter(projectId, name),
  writeCharacter(projectId, name, content), deleteCharacter(projectId, name),
  listSummaries(projectId), getSummary(projectId, chapterId),
  getFile(projectId, path), writeFile(projectId, path, content),
  getProjectMemory(projectId)
}
```

---

## 第 3 轮：回收站 + 用户中心 + 收尾

### 回收站
- 对接: `GET /trash`, `POST /trash/restore`, `DELETE /trash`
- 位置: 书架页或编辑器侧栏入口
- 功能: 列表 + 恢复 + 清空

### 用户中心
- 对接: `GET/PATCH /user/me`, `GET /user/usage`
- 位置: 新页面或弹窗
- 功能: 昵称/头像编辑 + AI 用量展示

### 项目重命名
- 对接: `PATCH /projects/:id`
- 位置: 书架页加编辑按钮

### 项目导出
- 对接: `GET /projects/:id/export`
- 功能: 下载 .md/.txt

### 密码重置
- 后端: 新增 1 个 handler (复用验证码流程, scene='reset_password')
- 前端: 登录页加「忘记密码」入口

### 配额恢复
- 文件: `gateway.go:60,157` + `middleware.go:47`
- 修复: 解开 3 处 TODO，tier 从 user.Plan 读取

---

## 26 个未对接 API 速查

| API | 归属轮次 |
|-----|---------|
| POST /auth/refresh | 第1轮 |
| GET/PATCH /user/me | 第3轮 |
| GET /user/usage | 第3轮 |
| GET /projects/search | 未规划(低优) |
| PATCH /projects/:id | 第3轮 |
| GET /projects/:id/export | 第3轮 |
| GET/POST/DELETE /trash (3个) | 第3轮 |
| /memory/* (12个) | 第2轮 |
| GET /ai/models | 未规划(低优) |

## 部署
- 测试服: 10.10.30.180, 服务 systemd=xiezuojun-server, nginx 3100→8081
- 编译: `go build -o xiezuojun-server ./cmd/server/`
- 前端: `npm run build` → dist 同步到 /opt/xiezuojun-web/dist/
- rsync 排除 .env
