export interface IMiniappJson {
    isMiniAppProject: (project?: IProject) => boolean;
    setIsMiniAppProject?: (func: FN) => void;
}
export declare const IMiniappJson: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IMiniappJson>;
