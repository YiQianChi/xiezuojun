# 写作菌 xiezuojun

AI 驱动的小说创作辅助 SaaS：项目管理、Tiptap 富文本编辑、AI 续写/改写/润色/扩写/大纲、三层记忆系统（角色卡/关系/章节摘要）、用量配额、DOCX 导出。

线上：**https://xiezuojun.com**（ICP 苏ICP备2026036211号-4 · 南京江北新区光继蓝网络科技）

## 仓库结构（monorepo）

```
server/   Go 1.21 + Gin + GORM + MySQL 后端（原 xiezuojun-server 仓库，历史保留）
web/      Vue 3 + TypeScript + Vite + Tiptap 3 前端（原 xiezuojun-web 仓库，历史保留）
docs/     项目级文档（部署、运维、架构地图）
```

> 2026-10-20 由两个独立仓库 git-filter-repo 合并而成，全部 93 个历史 commit 保留。

## 快速开始

### 后端

```bash
cd server
mysql -u root -p -e "CREATE DATABASE xiezuojun CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
cp .env.example .env   # 填数据库密码 + AI API Key
export $(cat .env | xargs) && go run ./cmd/server/
```

### 前端

```bash
cd web
npm install && npm run dev
```

### 测试账号（测试服 180）

`test0728@xiezuojun.com` / `Test123456`（项目 id=37）

## 文档索引

| 文档 | 内容 |
|------|------|
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | 生产/测试环境、部署流程 checklist、DB 与凭证 |
| [docs/PROJECT_MAP.md](docs/PROJECT_MAP.md) | 前后端 codebase 全景地图（目录、API 路由、数据流） |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 后端架构详解 |
| [docs/ROADMAP.md](docs/ROADMAP.md) | 路线图：已完成迭代记录 + 当前待办 |
| [docs/design/](docs/design/) | 设计方案：存储层重设计、关系图谱 |
| [server/README.md](server/README.md) | 后端 API 概览与环境变量 |
| [web/README.md](web/README.md) | 前端技术栈与目录速览 |

## 部署速览

- **生产**：124.222.138.209 · systemd `xiezuojun-server` · `/opt/xiezuojun-server`（二进制+.env）· 前端 dist rsync 到 `/opt/xiezuojun-web/dist/`
- **测试**：10.10.30.180 · DB `mysql -unimingben -p… xiezuojun`
- 详细步骤与铁律（改 .env 必须重启、生产写库先 SELECT 确认等）见 [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
