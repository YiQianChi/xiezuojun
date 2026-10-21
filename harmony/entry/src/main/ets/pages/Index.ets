// 写作菌 · HarmonyOS 壳主页面（带诊断日志版）
import { webview } from '@kit.ArkWeb';
import { hilog } from '@kit.PerformanceAnalysisKit';

const TAG = 'xzj-web';
const DOMAIN = 0x0001;

@Entry
@Component
struct Index {
  controller: webview.WebviewController = new webview.WebviewController();

  build() {
    // ponytail-debug: Web 直接作根组件（不经 Column 包裹），排除百分比高度塌陷导致白屏
    Web({ src: 'https://xiezuojun.com', controller: this.controller })
      .width('100%')
      .height('100%')
      .domStorageAccess(true)
      .databaseAccess(true)
      .fileAccess(true)
      .onlineImageAccess(true)
      .javaScriptAccess(true)
      .mixedMode(MixedMode.None)
      .cacheMode(CacheMode.Default)
      .multiWindowAccess(false)
      .onControllerAttached(() => {
        hilog.info(DOMAIN, TAG, 'controller attached');
      })
      .onPageBegin((e) => {
        hilog.info(DOMAIN, TAG, 'page begin: %{public}s', e?.url ?? '');
      })
      .onPageEnd((e) => {
        hilog.info(DOMAIN, TAG, 'page end: %{public}s', e?.url ?? '');
      })
      .onPageVisible((e) => {
        hilog.info(DOMAIN, TAG, 'page visible: %{public}s', e?.url ?? '');
      })
      .onErrorReceive((e) => {
        hilog.error(DOMAIN, TAG, 'load error: %{public}s', e?.request?.getUrl() ?? '');
      })
      .onConsole((e) => {
        hilog.warn(DOMAIN, TAG, 'console: %{public}s', e?.message?.getMessage() ?? '');
      })
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
