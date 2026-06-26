# xiezuojun-server 架构文档

## 1. 项目概述

**写作菌后端** — AI 驱动的小说创作辅助服务平台。提供项目管理、文档编辑、AI 续写/改写/润色/扩写/大纲生成、记忆系统（角色卡、章节摘要、三层记忆）、用量配额管理等功能。

- **语言/框架**: Go 1.21 + Gin + GORM
- **数据库**: MySQL（生产）/ SQLite（迁移工具）
- **AI 接口**: 智谱 GLM 系列 / 任意 OpenAI-compatible API（火山引擎）
- **认证**: JWT（access 2h + refresh 30d）
- **存储**: MySQL（元数据）+ 文件系统（文档正文 + 记忆文件）

## 2. 目录结构

```
cmd/
  server/main.go          # 入口：初始化 DB、路由、handler、启动服务
  migrate/                 # 数据迁移工具（DB→文件系统）
    main.go               # CLI 入口
    migrate_db.go         # DB 连接、统计
    migrate_docs.go       # 文档迁移
    migrate_memory.go     # 记忆迁移
internal/
  ai/                      # AI 客户端
    client.go             # OpenAI-compatible HTTP 客户端（流式 + 非流式）
    handler.go            # AI handler（ChatStream、GenerateOutline、PlotContinue）
  auth/jwt.go             # JWT 管理器
  config/config.go        # 环境变量配置加载
  gateway/                # AI 网关（核心编排层）
    types.go              # 核心类型定义（ContextBlock、ModeTemplate、ModelConfig、Quota 等）
    gateway.go            # DefaultGateway：编排上下文→路由→AI调用→用量记录
    context_builder.go    # 上下文金字塔构建器
    model_router.go       # 模型选择路由器
    usage_tracker.go      # 用量追踪与配额管理（内存+DB双写）
    api_handler.go        # HTTP handler（/ai/generate、/ai/models）
    middleware.go          # 认证 + 限流中间件
  memory/                 # 记忆系统
    storage.go            # MemoryStorage：文件系统操作
    layers.go             # 三层记忆（短期/长期/风格）
    chapters.go           # 章节摘要存储
    characters.go         # 角色卡 CRUD
    extractor.go          # AI 自动提取摘要/角色/事件/伏笔
    handler.go            # HTTP handler 基类
    handler_layers.go     # 三层记忆 API
    handler_files.go      # 通用记忆文件 API
    handler_characters.go # 角色卡 API
    handler_summaries.go  # 章节摘要 API
    handler_extract.go    # 记忆提取 API
  middleware/              # 通用中间件
    auth.go               # JWT 认证
    cors.go               # CORS
  model/                   # GORM 数据模型
    user.go               # User + RefreshToken
    project.go            # Project + Document + DocumentVersion
    usage.go              # WordCountLog
    usage_record.go       # UsageRecord（AI 用量）
  project/                 # 项目与文档管理
    handler.go            # 项目 CRUD + 导出
    doc_handler.go        # 文档列表/详情/删除
    doc_write_handler.go  # 文档创建/更新
    trash_handler.go      # 回收站
    storage.go            # DocStore：文档文件读写
    helpers.go            # 工具函数
  store/                   # 文件操作基础层
    path.go               # PathConfig + 路径安全化
    atomic.go             # 原子写入
    lock.go               # 用户级读写锁
    quota.go              # 字数配额
    trash.go              # 回收站操作
    changelog.go          # 变更日志
```

## 3. 核心数据模型

### User（users 表）
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | char(36) | UUID 主键 |
| Email | varchar(255) | 唯一索引 |
| Phone | varchar(20) | |
| Nickname | varchar(50) | |
| PasswordHash | varchar(255) | bcrypt |
| Avatar | varchar(500) | |
| Plan | varchar(20) | free / vip / svip |
| Status | int | 1=正常 0=禁用 |

### RefreshToken（refresh_tokens 表）
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 自增主键 |
| UserID | char(36) | 索引 |
| Token | varchar(512) | 唯一索引 |
| DeviceID | varchar(100) | |
| IP | varchar(45) | |
| ExpiresAt | time.Time | |

### Project（projects 表）
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 自增主键 |
| UserID | uint | 索引 |
| Name | varchar(200) | |
| CoverURL | varchar(500) | |
| Genre | varchar(50) | 小说/剧本/散文 |
| Tags | varchar(500) | 逗号分隔 |
| Status | int | 1=正常 0=删除 |

### Document（documents 表）
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 自增主键 |
| ProjectID | uint | 索引 |
| ParentID | *uint | 支持卷/章嵌套 |
| Type | varchar(20) | volume / chapter / note |
| Title | varchar(200) | |
| SortOrder | int | |
| WordCount | int | |
| Status | int | 1=正常 0=删除 |

### UsageRecord（usage_records 表）
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 自增主键 |
| UserID | varchar(50) | 索引 |
| ProjectID | varchar(50) | 索引 |
| ModelID | varchar(100) | |
| PromptTokens | int | |
| CompletionTokens | int | |
| TotalTokens | int | |
| Cost | float64 | |
| AIMode | varchar(20) | continue/rewrite/polish/expand/outline |
| DurationMs | int64 | |
| CreatedAt | time.Time | 索引 |

### 文件系统存储
```
{DATA_ROOT}/
  documents/{userID}/{projectID}-{projectName}/
    {docID}.md              # 文档正文
    structure.json          # 项目结构
    .trash/                 # 回收站
      {docID}.md
      {docID}.md.meta.json
  memory/{userID}/{projectID}/
    working_memory.md       # 短期记忆
    long_term.md            # 长期记忆
    STYLE.md                # 风格记忆
    PREFERENCES.md          # 作者偏好（强制注入）
    WORLD.md                # 世界观
    TIMELINE.md             # 事件时间线
    HOOKS.md                # 伏笔追踪
    characters/             # 角色卡
      {name}.md
    chapters/               # 章节摘要
      {docID}.summary.md
  changelog/{userID}.jsonl  # 变更日志
  quota/{userID}.json       # 字数配额
```

## 4. 数据流

### AI 生成流程（核心链路）
```
客户端 POST /api/ai/generate
  → middleware.JWTAuth()          # 验证 JWT，提取 userID
  → gateway.RateLimitMiddleware()  # 配额检查
  → gateway.HandleGenerate()       # HTTP handler
    → DefaultGateway.Generate() 或 .GenerateStream()
      1. CheckQuota()              # 日/月配额检查
      2. ContextBuilder.Build()    # 构建上下文金字塔
         - 读取 PREFERENCES.md（强制注入）
         - 读取 STYLE.md、WORLD.md
         - DetectCharacterNames() 从前文识别人名 → 加载角色卡
         - 加载前文（SceneText）
         - 按层级 token 上限截断
      3. ModelRouter.Route()       # 选择模型（按优先级+约束）
      4. acquireSlot()             # 用户级并发控制（默认 2）
      5. aiClient.Stream() / .Complete()  # 调用 AI API
      6. UsageTracker.Track()      # 记录用量（内存 + MySQL）
    → SSE 流式返回 / JSON 一次性返回
```

### 文档更新触发记忆提取
```
客户端 PATCH /api/documents/:id
  → main.go 中的包装 handler
    1. 读取请求体，恢复 body
    2. 调用 projectHandler.UpdateDocument()
    3. 如果成功（200），启动 goroutine：
       → memory.ExtractAndSaveSummary()
         a. AI 生成章节摘要 → chapters/{docID}.summary.md
         b. AI 提取角色/事件/地点/伏笔（JSON）
         c. 新角色 → 自动创建骨架角色卡
         d. 事件 → 追加到 TIMELINE.md
         e. 伏笔 → 追加到 HOOKS.md
```

## 5. 关键接口签名

### Gateway 接口（gateway/types.go）
```go
type Gateway interface {
    BuildContext(ctx, req BuildContextRequest) (*BuildContextResult, error)
    Generate(ctx, req GenerateRequest) (*GenerateResult, error)
    GenerateStream(ctx, req GenerateRequest) (<-chan StreamChunk, error)
    CheckQuota(userID, tier string) (*QuotaStatus, error)
    TrackUsage(record UsageRecord) error
}
```

### AI Client（ai/client.go）
```go
func (c *Client) Stream(ctx, model string, msgs []Message) (<-chan StreamMessage, error)
func (c *Client) Complete(ctx, model string, msgs []Message) (string, error)
```

### MemoryReader（gateway/context_builder.go）
```go
type MemoryReader interface {
    ReadCharacter(uid, pid, name string) (string, error)
    ReadMemoryFile(uid, pid, path string) (string, error)
}
```

### MemoryStorage（memory/storage.go）
```go
func (s *MemoryStorage) ReadWorkingMemory(uid, pid string) (string, error)
func (s *MemoryStorage) WriteWorkingMemory(uid, pid, content string) error
func (s *MemoryStorage) ReadLongTermMemory(uid, pid string) (string, error)
func (s *MemoryStorage) DistillMemory(uid, pid string) error  // 短期→长期蒸馏
func (s *MemoryStorage) ReadChapterSummary(uid, pid, docID string) (string, error)
func (s *MemoryStorage) ReadCharacter(uid, pid, name string) (string, error)
func (s *MemoryStorage) ReadMemoryFile(uid, pid, path string) (string, error)
```

## 6. 跨文件依赖

```
cmd/server/main.go
  ├→ internal/ai          # AI 客户端
  ├→ internal/auth         # JWT
  ├→ internal/config       # 配置
  ├→ internal/gateway      # AI 网关
  │    ├→ internal/ai      # 调用 AI API
  │    └→ internal/memory  # MemoryReader 接口由 MemoryStorage 实现
  ├→ internal/memory       # 记忆系统
  │    ├→ internal/ai      # 提取时调用 AI
  │    └→ internal/store   # 文件原子写入
  ├→ internal/middleware   # JWT 认证
  ├→ internal/model        # GORM 模型
  ├→ internal/project      # 项目/文档管理
  │    └→ internal/store   # 文件操作
  ├→ internal/store        # 存储基础层
  └→ internal/user         # 用户认证

internal/gateway/context_builder.go → internal/memory (DetectCharacterNames)
internal/memory/extractor.go → internal/ai (AI 调用)
cmd/server/main.go 中设置 gw.SetDocContentLoader() 连接 gateway ↔ project
```

## 7. 配置和环境变量

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `DB_DSN` | （无默认，必填） | MySQL DSN |
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥 |
| `AI_API_KEY` | `sk-test` | AI API 密钥 |
| `AI_BASE_URL` | `https://open.bigmodel.cn/api/paas/v4` | AI API 地址 |
| `PORT` | `8080` | 服务端口 |
| `DATA_ROOT` | `./data` | 数据根目录 |
| `DOC_ROOT` | `{DATA_ROOT}/documents` | 文档存储根目录 |
| `MEMORY_ROOT` | `{DATA_ROOT}/memory` | 记忆存储根目录 |

## 8. 踩坑记录

### userID 类型不一致
- `model.User.ID` 是 `string`（UUID），但 middleware/auth.go 中从 JWT claims 取出的 `userID` 是 string，部分 handler（project/helpers.go）用的是 `uint`。
- gateway/usage_tracker.go 中 `UserID` 字段为 `string`，但 gateway/gateway.go 实际传入了 `ProjectID` 作为 `UserID`（有 TODO 注释）。
- **注意**: 硬编码 tier 为 `"free"`，未从用户系统读取真实 Plan。

### token 估算精度
- `estimateTokens()` 使用简单中英文字符计数，与实际 tokenizer 结果差异可能达 20-30%。对长文本可能造成上下文截断过早或过晚。

### 并发控制
- `DefaultGateway.semaphores` 每个 userID 一个 channel，容量 2，但没有按 tier 区分并发限制。
- `releaseSlot()` 使用 `select { case <-sem: default: }` 非阻塞释放，如果 acquire 失败后 defer 仍会执行 release，可能造成计数错乱。

### 记忆提取异步
- `main.go` 中文档更新后通过 `go func()` 异步触发 `ExtractAndSaveSummary`，使用 `c.Request.Context()` — 如果 HTTP 请求已结束，context 可能被取消，导致 AI 调用失败。应使用 `context.Background()`。

### 文件路径安全
- `SanitizeName()` 过滤非法字符但保留空格，在路径拼接时可能产生意外行为。

## 9. 变更日志

| 日期 | 变更 |
|------|------|
| 2026-05-13 | 实施 6 项匹配机制调整（用户/项目维度） |
| 2026-06-07 | 生产环境排查可疑房间（确认非爬虫） |
| 2026-06-26 | 生成 ARCHITECTURE.md 架构文档 |
