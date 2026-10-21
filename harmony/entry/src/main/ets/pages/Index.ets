// 写作菌 · HarmonyOS 壳主页面
// 策略：前端 dist 打包在 rawfile/web/ 本地加载（秒开、离线有壳），
// API 请求（/api/*）通过 Web 组件的 onInterceptRequest 逻辑在前端侧走绝对地址——
// 因此前端构建时用 VITE_API_BASE 注入 https://xiezuojun.com（见 web/.env.harmony）。
// 这里只负责：全屏 WebView + 沉浸式 + 返回键处理（网页内后退而非退出应用）。

import { webview } from '@kit.ArkWeb';
import { AsyncCallback } from '@kit.BasicServicesKit';

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
        .darkMode(WebDarkMode.Auto)      // 跟随系统深浅色（前端已适配 .dark）
        .verticalScrollBarAccess(false)  // 移动端靠触摸滚动
        .multiWindowAccess(false)
        .onBackPress(() => {
          // 系统返回键：优先让网页内部后退（SPA 路由），历史栈空了才退出应用
          if (this.controller.accessBackward()) {
            this.controller.backward();
            return true;
          }
          return false;
        })
        .onPageBegin(() => {
          // 注入 viewport 兜底（rawfile 里的 index.html 已有，这里双保险）
          this.controller.runJavaScript(
            'var m=document.querySelector(\'meta[name=viewport]\');' +
            'if(!m){m=document.createElement(\'meta\');m.name=\'viewport\';' +
            'm.content=\'width=device-width,initial-scale=1.0\';document.head.appendChild(m);}'
          ).catch((_e: string) => {});
        })
        .onErrorReceive((event) => {
          if (event && !event.request.isMainFrame()) {
            return; // 子资源失败（如离线时 API 请求）不打断页面
          }
        })
    }
    .width('100%')
    .height('100%')
    .expandSafeArea({ edges: [{ type: SafeAreaType.System, edges: [SafeAreaEdge.Top, SafeAreaEdge.Bottom] }] })
  }
}
