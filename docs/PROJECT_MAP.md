# 注意

> 本文目录路径写的是合并前的两仓结构：`/root/workspace/projects/xiezuojun-server` 现对应本仓库 `server/`，`xiezuojun-web` 对应 `web/`。内部相对路径（internal/、src/ 等）仍然准确。

# 写作菌 Codebase Map

> 供 subagent 快速定位代码。配合 `search_files` 和 `read_file` 使用。
> 过时了跑 `repomix --include "internal/**/*.go" --output /tmp/xiezuojun-backend-map.txt` 重新提取。

## 部署架构

- **后端源码**: `/root/workspace/projects/xiezuojun-server` (Go + Gin + GORM + MySQL)
- **后端源码**: `/root/workspace/projects/xiezuojun-server` (Go + Gin + GORM + MySQL)
- **生产后端**: `124.222.138.209` (xiezuojun.com), systemd unit `xiezuojun-server`
  - ExecStart=`/opt/xiezuojun-server/xiezuojun-server`, EnvironmentFile=`/opt/xiezuojun-server/.env`, PORT=8081
  - 改 .env 前先备份 (`sudo cp .env .env.bak.$(date +%Y%m%d_%H%M%S)`)；改完必须 `sudo systemctl restart xiezuojun-server`（ubuntu 用户必须 sudo；env 只在进程启动时读，不重启不生效）
  - 本机 /opt/xiezuojun-server* 只是旧拷贝，不是生产
- **前端源码**: `/root/workspace/projects/xiezuojun-web` (Vue3 + TS + Tiptap 3)
- **生产前端**: rsync dist/ 到 `/opt/xiezuojun-web/dist/`
- **测试服**: `10.10.30.180`, DB在本地
- **DB**: `127.0.0.1:3306`, `xiezuojun:Xzj2026Test@tcp(127.0.0.1:3306)/xiezuojun` (生产本地)
- **测试账号**: test0728@xiezuojun.com / Test123456, 项目id=37
- **短信**: 火山引擎, SmsAccount=8bb50d20, 注册模板S1T_1y2pbauk98v0i/改密模板S1T_1y2pbauk98vlu

## AI 供应商配置（换key/换模型看这里）

- key和endpoint在 `/opt/xiezuojun-server/.env`：`AI_API_KEY` + `AI_BASE_URL`（OpenAI兼容格式）
- 模型名**不在.env**：生成走 `internal/gateway/types.go` 的 `DefaultModels`（硬编码 ModelID，改动需重新build部署）；记忆抽取走 `internal/memory/extractor.go` 的 `memoryModel()`（env链 MEMORY_MODEL→AI_MODEL→默认 MiniMax-M3）
- 火山引擎两个端点别混：`api/plan/v3`=通用套餐；`api/coding/v3`=coding套餐专属，key不匹配/过期直接404
- 可用ark key优先从匿名本生产拿（共用额度，Boss知情）：ssh 122 → `mysql -h10.1.0.15 -unimingben -pNb2026Admin nimingben -e "SELECT id,name,model,endpoint,api_key FROM bot_providers WHERE enabled=1"`
- 换key前先curl实测（模型名用 DefaultModels 里的）：
  `curl -X POST $AI_BASE_URL/chat/completions -H "Authorization: Bearer $KEY" -H 'Content-Type: application/json' -d '{"model":"MiniMax-M3","messages":[{"role":"user","content":"说OK"}],"max_tokens":10}'`

## 后端目录结构 (Go)

```
cmd/server/main.go  — 入口, 路由注册
internal/
  ai/         — AI客户端封装: client.go(多provider), handler.go
  auth/       — JWT: jwt.go
  gateway/    — AI网关核心:
    gateway.go          — Generate()/GenerateStream() 入口, 并发槽位管理(acquireSlot/releaseSlot)
    api_handler.go      — HTTP: HandleGenerate, HandleGetModels
    chat_handler.go     — SSE流式对话: HandleChat, chatComplete, chatStream, loadChatContext, trackChatUsage
    context_builder.go  — 上下文组装: Build(), collectBlocks(), truncateBlocks(), buildSystemPrompt()
    model_router.go     — 模型路由(选择provider)
    middleware.go       — 网关中间件
    types.go            — 类型定义
    usage_tracker.go    — 用量追踪
  memory/     — AI长期记忆系统(小说世界观):
    handler_characters.go, handler_relationships.go, handler_summaries.go, handler_files.go, handler_layers.go
    chapters.go, characters.go, relationships.go, layers.go, summaries.go
    extractor.go(信息抽取), storage.go, handler.go
  middleware/ — auth.go, cors.go
  model/      — GORM模型: project, user, usage, usage_record
  project/    — 项目文档管理(核心):
    handler.go            — 文档CRUD入口
    doc_handler.go        — 文档读写
    doc_write_handler.go  — 文档写入(含AI结果应用)
    doc_version_handler.go — 文档版本
    doc_move_handler.go   — 文档移动/排序
    branch_handler.go     — 分支管理
    thread_handler.go     — 故事线
    export_handler.go     — 导出
    search_handler.go     — 全文搜索
    search_rebuild.go     — 搜索重建索引
    stats_handler.go      — 统计
    trash_handler.go      — 回收站
    storage.go            — 文件存储
    default_structure.go  — 默认项目结构
  store/      — 存储引擎:
    atomic.go/atomic_test.go — 原子写入
    lock.go    — 文件锁
    path.go    — 路径管理
    quota.go   — 配额
    trash.go   — 回收站
    changelog.go — 变更日志
  user/       — 用户: handler.go(注册/登录), email_code.go(验证码)
```

## 核心 API 路由

```
/api/health                      GET   健康检查
/api/auth/email-code             POST  发送验证码
/api/auth/register               POST  注册
/api/auth/login                  POST  登录
/api/projects                    CRUD  项目管理
/api/projects/:id/docs           CRUD  文档管理
/api/projects/:id/docs/:docId/versions — 文档版本
/api/projects/:id/branches       CRUD  分支
/api/projects/:id/threads        CRUD  故事线
/api/projects/:id/export         POST  导出
/api/projects/:id/trash          CRUD  回收站
/api/generate                    POST  AI生成(非流式)
/api/generate/stream             POST  AI生成(SSE流式)
/api/chat                        POST  AI对话(SSE流式)
/api/models                      GET   可用模型列表
/api/memory/*                    CRUD  AI记忆(角色/关系/摘要/文件/图层)
```

## AI 生成数据流

```
请求 → gateway/api_handler.go (HandleGenerate)
     → gateway/gateway.go (Generate/GenerateStream)
       → acquireSlot() — 并发槽位控制
       → model_router.go — 选模型/provider
       → context_builder.go Build() — 组装上下文:
         collectBlocks() — 收集文档内容/设置/线程
         truncateBlocks() — 按token预算截断
         buildSystemPrompt() — 系统提示词
       → ai/client.go — 调用LLM
       → usage_tracker.go — 记录用量
     → 释放槽位
```

## 文档存储模型

```
project/ (MySQL: projects表)
  ├── documents (JSON文件存储, 非MySQL)
  │   ├── doc版本 (DocumentVersion, MySQL)
  │   ├── 分支 (StoryBranch, MySQL)
  │   └── 故事线 (StoryThread, MySQL)
  └── memory/ (AI记忆, MySQL)
      ├── characters (角色)
      ├── relationships (关系)
      ├── summaries (摘要)
      └── layers (图层)
```

## DB 模型

```
Project        — id, user_id, title, created_at
Document       — id, project_id, title, content, parent_id, sort_order
DocumentVersion — id, doc_id, content, version_note, created_at
StoryBranch    — id, project_id, name, parent_branch_id
StoryThread    — id, project_id, title, content
User           — id, email, password, created_at
UsageRecord    — id, user_id, project_id, model, prompt_tokens, completion_tokens, cost
WordCountLog   — 写作字数统计
```

## 前端目录结构 (Vue3 + Tiptap 3)

```
src/
  components/
    editor/     — 编辑器面板:
      EditorSidebar.vue     — 文档树侧边栏
      EditorHeader.vue      — 顶部栏
      AIChatPanel.vue       — AI对话
      AIContextPanel.vue    — AI上下文设置
      AIChatPanel.vue       — AI聊天面板
      MemoryPanel.vue       — 记忆管理(角色/关系)
      ReferencePanel.vue    — 参考资料
      RelationshipGraph.vue — 关系图谱
      StoryboardView.vue    — 故事板
      ThreadPanel.vue       — 故事线面板
      TrashPanel.vue        — 回收站
      VersionHistoryModal.vue — 版本历史
      WritingStatsPanel.vue — 写作统计
      WritingTimer.vue      — 写作计时
      FolderPropsModal.vue  — 文件夹属性
      CreateDocModal.vue    — 新建文档
      SaveVersionModal.vue  — 保存版本
    DocTreeNode.vue   — 文档树节点(递归, 支持拖拽)
    OnboardingTour.vue — 引导
    PlanView.vue      — 大纲视图
    ReviewView.vue    — 审阅视图
    SearchPanel.vue   — 搜索面板
  composables/
    useAIToolbar.ts      — AI工具栏逻辑
    useEditorSearch.ts   — 编辑器搜索
    useToast.ts          — 提示
    useVersionManager.ts — 版本管理
  core/
    api.ts  — API请求封装
    sse.ts  — SSE流式连接
  pages/
    Bookshelf.vue — 书架(项目列表)
    Editor.vue    — 编辑器(主页面)
    Home.vue      — 首页
    Login.vue     — 登录
    Register.vue  — 注册
  router/index.ts
```

## 前端关键约定

- 2026-06-30 从零重写，Vue3 + Tiptap 3 + TS
- AI 编辑动作(改写/续写/润色/扩写)的预览交互: 居中卡片(原文删除线 + AI结果可改 + 采用/丢弃)
- 文档树使用 DocTreeNode.vue 递归组件, 支持拖拽排序
- SSE 流式: core/sse.ts 连接后端 /api/generate/stream
