# 写作菌存储架构重构 — 设计文档

## 目标目录结构

```
/opt/xiezuojun/
├── documents/{user_id}/          # 用户文件
│   ├── .profile.json             # 用户偏好
│   ├── .quota.json               # 用量配额
│   ├── .changelog                # 操作日志（append-only）
│   │
│   └── {sanitized_project_name}/ # 书名目录
│       ├── .meta.json            # 书籍元信息
│       ├── .structure.json       # 文档树（标题/类型/排序/父子关系）
│       ├── .trash/               # 回收站（7天自动清理）
│       │   └── 11.meta.json
│       ├── 10.md                 # 正文（ID命名）
│       ├── 11.md
│       └── 15.md
│
└── memory/{user_id}/             # AI 记忆系统（独立）
    └── {project_id}/
        ├── working_memory.md
        ├── long_term.md
        ├── STYLE.md
        ├── CHARACTERS/
        └── chapters/
```

## 核心变更

### 1. 新建 `internal/store/` 包

文件操作基础层，所有上层存储都依赖它。

#### `internal/store/path.go` — 路径计算
```go
type PathConfig struct {
    DocRoot    string // /opt/xiezuojun/documents
    MemRoot    string // /opt/xiezuojun/memory
}

// SanitizeName 书名/文件名安全化
// 过滤 / \ .. < > | * ? " '，截断50字符，空名默认 untitled-{id}
func SanitizeName(name string, fallbackID uint) string

// UserDir 返回用户根目录
func (p *PathConfig) UserDir(userID uint) string
// ProjectDir 返回项目目录
func (p *PathConfig) ProjectDir(userID, projectID uint, projectName string) string
// DocPath 返回文档正文路径
func (p *PathConfig) DocPath(userID, projectID uint, projectName string, docID uint) string
// MemoryDir 返回记忆目录
func (p *PathConfig) MemoryDir(userID, projectID uint) string
```

#### `internal/store/atomic.go` — 原子写入
```go
// AtomicWrite 原子写入：写 tmp → fsync → rename
func AtomicWrite(path string, data []byte, perm os.FileMode) error

// AtomicWriteJSON 原子写 JSON
func AtomicWriteJSON(path string, v any) error

// AtomicAppend 追加写入（用于 changelog）
func AtomicAppend(path string, data []byte) error
```

#### `internal/store/lock.go` — 并发写保护
```go
// UserLocks 按 userID 分组的读写锁池
type UserLocks struct {
    locks sync.Map // uint → *sync.RWMutex
}
func (ul *UserLocks) RLock(userID uint)
func (ul *UserLocks) RUnlock(userID uint)
func (ul *UserLocks) Lock(userID uint)
func (ul *UserLocks) Unlock(userID uint)
```

#### `internal/store/quota.go` — 配额管理
```go
type Quota struct {
    TotalWords    int `json:"total_words"`
    TotalFiles    int `json:"total_files"`
    MaxWords      int `json:"max_words"`       // free=1000000, VIP=0(不限)
}

// LoadQuota 读取用户配额
func LoadQuota(paths *PathConfig, userID uint) (*Quota, error)
// CheckQuota 检查是否超限，超限返回 error
func (q *Quota) CheckQuota(addWords int) error
// UpdateQuota 更新配额统计（原子写入）
func (q *Quota) Save(paths *PathConfig, userID uint) error
// RecalcQuota 重新计算（遍历用户目录）
func RecalcQuota(paths *PathConfig, userID uint) (*Quota, error)
```

#### `internal/store/changelog.go` — 变更日志
```go
type ChangeEntry struct {
    Time    string `json:"time"`
    Action  string `json:"action"`   // create_doc, update_doc, delete_doc, create_project, ...
    Project string `json:"project"`
    Doc     string `json:"doc"`
    Summary string `json:"summary"`
}

// AppendChangelog 追加变更记录
func AppendChangelog(paths *PathConfig, userID uint, entry ChangeEntry) error
```

#### `internal/store/trash.go` — 回收站
```go
// TrashMove 移入回收站
func TrashMove(docPath string, meta interface{}) error
// TrashRestore 从回收站恢复
func TrashRestore(trashPath string, destPath string) error
// TrashCleanup 清理超过7天的回收站文件
func TrashCleanup(paths *PathConfig, userID uint) (int, error)
// TrashList 列出回收站内容
func TrashList(paths *PathConfig, userID uint) ([]string, error)
```

### 2. 改造 `internal/project/` 包

#### `storage.go` 重写
- 从平铺文件 `data/documents/{id}.txt` 改为用户目录 `documents/{uid}/{project}/{id}.md`
- 所有文件操作走 `store.AtomicWrite`
- 写操作加 `UserLocks`
- 删除走 `store.TrashMove` 而不是只改数据库 status
- 每次写/删操作追加 changelog

#### handler.go 改动
- `CreateProject`: 创建用户目录 + 项目目录 + .meta.json
- `CreateDocument`: 写正文到 {project}/{id}.md，更新 .structure.json
- `UpdateDocument`: 原子写入正文，更新 .structure.json
- `DeleteDocument`: 移到 .trash/
- `GetDocument`: 从 .structure.json 读元信息，从 {id}.md 读正文
- `ListDocuments`: 从 .structure.json 读文档树（不再只依赖 MySQL）
- 新增 `RestoreDocument`: 从 .trash/ 恢复
- 新增 `EmptyTrash`: 清空回收站

#### MySQL 保留策略
- 暂时保留所有 MySQL 表做索引查询
- handler 优先从文件系统读取，MySQL 做快速查询的补充
- 以后可逐步去除 MySQL

### 3. 记忆系统迁移

#### `internal/memory/storage.go` 改造
- 从 `data/users/{uid}/projects/{pid}/memory/` 改为 `memory/{uid}/{pid}/`
- 所有写入走 `store.AtomicWrite`
- 路径计算走 `PathConfig.MemoryDir`

### 4. main.go 改动
- PathConfig 初始化（DOC_ROOT + MEMORY_ROOT 环境变量）
- store 包的注入
- 回收站定时清理 goroutine（每小时一次）

### 5. 迁移脚本 `cmd/migrate/main.go`
- 读取 MySQL 现有 projects + documents
- 读取 `data/documents/*.txt` 旧文件
- 按 user_id → project_id 组织到新目录结构
- 生成 .meta.json 和 .structure.json
- 验证：新旧对比 MD5
- 完成后打印统计，不自动删旧文件

## 代码规范
- 每个文件 ≤ 150 行
- 每个函数 ≤ 30 行
- 错误处理：返回 error，不 panic（除非 recover 包裹）
- JSON 结构定义放在对应文件顶部
- 注释用中文
