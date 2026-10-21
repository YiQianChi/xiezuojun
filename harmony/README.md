# 写作菌 · HarmonyOS 壳工程

ArkWeb 壳：本地打包前端 dist + API 走 https://xiezuojun.com。与文舟同款路线，但我们的壳更薄——没有插件系统/本地 TeX，只有 WebView + 网络权限 + 分享。

## 用 DevEco Studio 打开

1. DevEco Studio（API 12+ SDK）→ Open → 选本目录 `harmony/`
2. 首次会提示签名配置：File → Project Structure → Signing Configs → 勾选 Automatically generate signature（调试签名，用你的华为账号登录即可）
3. 连接 HarmonyOS 手机/模拟器 → Run ▶

## 目录结构

```
harmony/
├── AppScope/app.json5            # 应用级配置（bundleName、图标）
├── build-profile.json5           # 工程级构建配置（SDK 版本、签名）
├── entry/
│   ├── src/main/
│   │   ├── ets/entryability/     # UIAbility 入口
│   │   │   └── EntryAbility.ets
│   │   ├── ets/pages/Index.ets   # ArkWeb 主页面
│   │   ├── resources/            # 图标、字符串
│   │   │   └── rawfile/web/      # ← 前端构建产物放这里（见下）
│   │   └── module.json5          # 模块配置（权限声明）
│   └── build-profile.json5
├── hvigor/                       # 构建工具配置
├── hvigorfile.ts
└── oh-package.json5
```

## 更新前端产物

每次前端改版后，重新把 `web/dist/*` 拷进 `entry/src/main/resources/rawfile/web/`：

```bash
# 在仓库根目录
rm -rf harmony/entry/src/main/resources/rawfile/web
cp -r web/dist harmony/entry/src/main/resources/rawfile/web
```

壳内加载策略：`rawfile/web/index.html` 本地渲染，**API 请求走 `https://xiezuojun.com`**（前端代码里 axios baseURL 是相对路径 `/api`，ArkWeb 里用 `onInterceptRequest` 把 `/api/*` 转发到线上域名——见 Index.ets 内注释）。

## 上架检查清单（提交华为市场前）

- [ ] bundleName 改成正式的（`com.xiezuojun.app` 之类，注册开发者账号后定）
- [ ] 隐私政策 URL（可用 xiezuojun.com/privacy，页面还没做）
- [ ] 首次启动隐私弹窗（同意后才进应用——华为审核硬要求）
- [ ] MIIT App 备案号（华为市场 2024 起强制）
- [ ] 应用图标/截图/简介材料
- [ ] 签名证书：发布证书（调试证书只能装机 100 台）

## 已知限制（当前壳版本）

- 无推送、无分享，纯 WebView 应用
- 离线时只显示缓存壳（首包本地，API 不可用）——和 PWA 一致
