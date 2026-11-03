# 部署与运维手册

> 权威服务器信息总表：182 机 `/root/servers.md`。本文只覆盖写作菌相关。

## 环境拓扑

| 角色 | 地址 | 登录 | 说明 |
|------|------|------|------|
| 开发机 | 本机（Hermes 宿主 183） | - | 代码 canonical `/root/workspace/projects/xiezuojun`，改完 push 再部署 |
| 旧开发机 | 10.10.30.182 | root/Le199789 (sshpass) | 历史源码树参考；hermes default agent 所在机 |
| 测试服 | 10.10.30.180 | root（公钥直连） | 前端 dist rsync 到 `/opt/xiezuojun-web/dist/`；DB `mysql -unimingben -pNb2026Admin xiezuojun` |
| 生产 | 124.222.138.209 | ubuntu（systemctl 需 sudo） | xiezuojun.com |

## 生产环境（124.222.138.209）

- **后端**：systemd unit `xiezuojun-server`
  - ExecStart=`/opt/xiezuojun-server/xiezuojun-server`，EnvironmentFile=`/opt/xiezuojun-server/.env`，PORT=8081
  - `.env` 关键项：`AI_API_KEY` + `AI_BASE_URL`（OpenAI 兼容格式）
  - **改 .env 前先备份**：`sudo cp .env .env.bak.$(date +%Y%m%d_%H%M%S)`
  - **改完必须重启才生效**（env 只在进程启动时读）：`sudo systemctl restart xiezuojun-server`
- **前端**：rsync `web/dist/` → `/opt/xiezuojun-web/dist/`
- **SEO 静态页**：`/opt/xzj-seo`，爬虫 UA 分流，独立于 dist（防 rsync 覆盖）；nginx 备份 `~/nginx-bak-xiezuojun.com.*.conf`
- **DB**（生产本地）：`127.0.0.1:3306`，`xiezuojun:Xzj2026Test@tcp(127.0.0.1:3306)/xiezuojun`
- **测试账号**：test@oneon.com.cn（昵称「审核体验」，小程序提审用；密码在 mp 后台测试账号设置，183 不留明文）。旧 test0728 账号已删
- **短信**：火山引擎 SmsAccount=8bb50d20；注册模板 S1T_1y2pbauk98v0i，改密模板 S1T_1y2pbauk98vlu；测试手机 17749538040
- **ICP**：苏ICP备2026036211号-4

## 部署流程（checklist）

**后端：**
1. 本机改 `server/` 代码 → `go build`（可 md5 对比生产二进制确认基线）
2. `git diff` 审查 → commit → push GitHub
3. scp 二进制到 `124:/opt/xiezuojun-server/xiezuojun-server`（**运行中不能直接 cp 覆盖，报 Text file busy**；要 `cp 到 .new 再 mv -f` 原子替换）
4. `sudo systemctl restart xiezuojun-server`
5. 验证：`curl http://127.0.0.1:8081/api/health`（在 124 上）+ 功能实测

**手机号/微信登录**（2026-11 上线）：`/auth/wx-phone-login`（小程序 getPhoneNumber 一键登录+孤儿号合并）、`/auth/sms-code`+`/auth/sms-login`（短信验证码，腾讯云，`TENCENT_SMS_*` 五项未配则 501）。微信侧只需 `WX_MP_APPID/WX_MP_APPSECRET`（已在 .env）。合并规则：手机号命中老账号则微信孤儿号（无其他登录方式）作品随迁+禁用；孤儿号有其他登录方式返回 409。

**前端：**
1. 本机 `cd web && npm run build`
2. rsync `dist/` → 测试服 180 验证 → 再 rsync 到 124 `/opt/xiezuojun-web/dist/`
3. 注意：SEO 静态页在 `/opt/xzj-seo`，别覆盖

## AI 供应商（换 key/换模型看这里）

- key 和 endpoint 在生产 `.env`：`AI_API_KEY` + `AI_BASE_URL`
- 模型名**不在 .env**：
  - 生成走 `server/internal/gateway/types.go` 的 `DefaultModels`（硬编码 ModelID，改动需重新 build 部署）
  - 记忆抽取走 `server/internal/memory/extractor.go` 的 `memoryModel()`（env 链 MEMORY_MODEL→AI_MODEL→默认 MiniMax-M3）
- 火山引擎端点：`api/plan/v3`=通用套餐；`api/coding/v3`=coding 套餐专属，key 不匹配/过期直接 404
- 换 key 前先 curl 实测
- 历史：2026-10-13 换为匿名本共用 ark key（plan/v3），旧 coding key 已 404

## 铁律

1. 生产 UPDATE/DELETE 前先 SELECT 列影响范围，等 Boss 确认；部署后必须 SELECT 验证
2. UI 改动先出 HTML demo（/tmp）多方案对比确认再改代码
3. 生产事故先止血再查根因
4. `.env` 改动不重启不生效

## 数据目录规范（2026-11-02 定）

所有持久化用户数据**一律放 DATA_ROOT 下按类型分子目录**，禁止在程序目录或 /opt 其他位置另起炉灶：

| 目录 | 内容 | 访问方式 |
|---|---|---|
| DATA_ROOT/documents/ | 书正文（按用户/书分层） | 登录态 API |
| DATA_ROOT/memory/ | 记忆/设定 | 登录态 API |
| DATA_ROOT/recycleBin/ | 回收站 | 登录态 API |
| DATA_ROOT/covers/ | 书封面（{书id}-{随机段}.png 平铺） | GET /api/covers/:file 免登录 |
| DATA_ROOT/avatars/（预留） | 用户头像 | 同封面模式 |

生产 DATA_ROOT=/opt/data，180 DATA_ROOT=/opt/xiezuojun-data。旧的 /opt/xiezuojun-server/data 空壳已删（早期默认路径残留，别再被它骗）。
