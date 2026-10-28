export interface IProjectManagerService {
    getAppIdAndAttr: () => Promise<Record<string, any>>;
    setGetAppIdAndAttr?: (func: FN) => void;
    getCurrentRuntimeConfig: () => IProjectAttr;
    setGetCurrentRuntimeConfig?: (func: FN) => void;
}
export declare const IProjectManagerService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IProjectManagerService>;
