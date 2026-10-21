// 写作菌 · HarmonyOS 壳主页面
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
        .domStorageAccess(true)
        .databaseAccess(true)
        .fileAccess(false)
        .onlineImageAccess(true)
        .javaScriptAccess(true)
        .mixedMode(MixedMode.None)
        .cacheMode(CacheMode.Default)
        .verticalScrollBarAccess(false)
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
