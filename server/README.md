# 写作菌后端 (xiezuojun-server)

写作菌 SaaS 后端服务，Go + Gin + GORM + MySQL。

## 快速开始

```bash
# 1. 创建数据库
mysql -u root -p -e "CREATE DATABASE xiezuojun CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 填入数据库密码和 AI API Key

# 3. 运行
export $(cat .env | xargs)
go run ./cmd/server/
```

## API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 注册 |
| POST | /api/auth/login | 登录 |
| POST | /api/auth/refresh | 刷新 Token |
| GET | /api/user/me | 当前用户 |
| PATCH | /api/user/me | 更新资料 |
| GET | /api/user/usage | 字数统计 |
| GET | /api/projects | 项目列表 |
| POST | /api/projects | 创建项目 |
| PATCH | /api/projects/:id | 更新项目 |
| DELETE | /api/projects/:id | 删除项目 |
| GET | /api/documents?project_id=1 | 文档列表 |
| POST | /api/documents | 创建文档 |
| GET | /api/documents/:id | 文档详情 |
| PATCH | /api/documents/:id | 更新文档 |
| DELETE | /api/documents/:id | 删除文档 |
| POST | /api/ai/chat | AI 对话（SSE） |
| POST | /api/ai/outline | 生成大纲 |
| POST | /api/ai/plot | 续写/润色（SSE） |

## 部署

```bash
# 编译 Linux 二进制
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o xiezuojun-server ./cmd/server/

# 服务器上
./xiezuojun-server
# 或用 systemd 管理
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| DB_DSN | MySQL 连接串 | - |
| JWT_SECRET | JWT 密钥 | change-me... |
| AI_API_KEY | 智谱AI API Key | - |
| AI_BASE_URL | AI API 地址 | 智谱官方 |
| PORT | 服务端口 | 8080 |
| DATA_ROOT | 文档存储根目录 | ./data |
