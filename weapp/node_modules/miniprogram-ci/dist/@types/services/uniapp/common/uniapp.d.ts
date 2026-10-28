export interface IUniApp {
    setCopyMiniAppJson?: (func: FN) => void;
    copyMiniAppJson: (project: IProject) => boolean;
}
export declare const IUniApp: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IUniApp>;
