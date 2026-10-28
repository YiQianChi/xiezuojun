export declare enum EInitializeStepsStatus {
    initializing = "initializing",
    ready = "ready",
    failed = "failed"
}
export declare enum EInitializeSteps {
    BackendMessagecenterDone = "BackendMessagecenterDone",
    RuntimeMessagecenterDone = "RuntimeMessagecenterDone",
    EditorMessagecenterDone = "EditorMessagecenterDone",
    BackendHasWorker = "BackendHasWorker",
    BackendSummerWorkerLoaded = "BackendSummerWorkerLoaded",
    BackendTaskWorkerLoaded = "BackendTaskWorkerLoaded",
    BackendWCC = "BackendWCC",
    BackendWCSC = "BackendWCSC",
    DefaultCompileInFirstTime = "DefaultCompileInFirstTime",
    RuntimeServiceWorker = "RuntimeServiceWorker"
}
export interface IInitializeCheck {
    notifiedStatus(step: EInitializeSteps, status: EInitializeStepsStatus): void;
}
export declare const IInitializeCheck: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IInitializeCheck>;
