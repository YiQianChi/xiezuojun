if (!("finalizeConstruction" in ViewPU.prototype)) {
    Reflect.set(ViewPU.prototype, "finalizeConstruction", () => { });
}
interface Index_Params {
    controller?: webview.WebviewController;
}
import webview from "@ohos:web.webview";
class Index extends ViewPU {
    constructor(parent, params, __localStorage, elmtId = -1, paramsLambda = undefined, extraInfo) {
        super(parent, __localStorage, elmtId, extraInfo);
        if (typeof paramsLambda === "function") {
            this.paramsGenerator_ = paramsLambda;
        }
        this.controller = new webview.WebviewController();
        this.setInitiallyProvidedValue(params);
        this.finalizeConstruction();
    }
    setInitiallyProvidedValue(params: Index_Params) {
        if (params.controller !== undefined) {
            this.controller = params.controller;
        }
    }
    updateStateVars(params: Index_Params) {
    }
    purgeVariableDependenciesOnElmtId(rmElmtId) {
    }
    aboutToBeDeleted() {
        SubscriberManager.Get().delete(this.id__());
        this.aboutToBeDeletedInternal();
    }
    private controller: webview.WebviewController;
    initialRender() {
        this.observeComponentCreation2((elmtId, isInitialRender) => {
            Column.create();
            Column.width('100%');
            Column.height('100%');
        }, Column);
        this.observeComponentCreation2((elmtId, isInitialRender) => {
            Web.create({ src: { "id": 0, "type": 30000, params: ['web/index.html'], "bundleName": "com.xiezuojun.app", "moduleName": "entry" }, controller: this.controller });
            Web.width('100%');
            Web.height('100%');
            Web.domStorageAccess(true);
            Web.databaseAccess(true);
            Web.fileAccess(false);
            Web.onlineImageAccess(true);
            Web.javaScriptAccess(true);
            Web.mixedMode(MixedMode.None);
            Web.cacheMode(CacheMode.Default);
            Web.verticalScrollBarAccess(false);
            Web.multiWindowAccess(false);
        }, Web);
        Column.pop();
    }
    // 系统返回键：网页内后退，历史栈空了才退出
    onBackPress(): boolean {
        if (this.controller.accessBackward()) {
            this.controller.backward();
            return true;
        }
        return false;
    }
    rerender() {
        this.updateDirtyElements();
    }
    static getEntryName(): string {
        return "Index";
    }
}
registerNamedRoute(() => new Index(undefined, {}), "", { bundleName: "com.xiezuojun.app", moduleName: "entry", pagePath: "pages/Index", pageFullPath: "entry/src/main/ets/pages/Index", integratedHsp: "false", moduleType: "followWithHap" });
