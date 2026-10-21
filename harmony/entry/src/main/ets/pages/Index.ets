// 写作菌 · HarmonyOS 壳主页面
// 策略：前端 dist 打包在 rawfile/web/ 本地加载（秒开、离线有壳），
// API 请求在前端侧走绝对地址——构建时用 VITE_API_BASE 注入 https://xiezuojun.com（见 web/.env.harmony）。
// 这里只负责：全屏 WebView + 返回键处理（网页内后退而非退出应用）。

import { webview } from '@kit.ArkWeb';

@Entry
@Component
struct Index {
  controller: webview.WebviewController = new webview.WebviewController();

  build() {
    Column() {
      Web({ src: $rawfile('web/index.html'), controller: this.controller })
        .width('100%')
        .height('100%')
        .domStorageAccess(true)          // localStorage：登录态/草稿快照/文档树缓存都靠它
        .databaseAccess(true)
        .fileAccess(false)               // 安全：不给 file:// 通用访问
        .onlineImageAccess(true)
        .javaScriptAccess(true)
        .mixedMode(MixedMode.None)       // API 全 https，不降级
        .cacheMode(CacheMode.Default)
        .verticalScrollBarAccess(false)  // 移动端靠触摸滚动
        .multiWindowAccess(false)
    }
    .width('100%')
    .height('100%')
  }

  // 系统返回键：网页内后退，历史栈空了才退出
  onBackPress(): boolean {
    if (this.controller.accessBackward()) {
      this.controller.backward();
      return true;
    }
    return false;
  }
}
