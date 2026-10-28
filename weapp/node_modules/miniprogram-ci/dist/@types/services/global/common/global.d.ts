export type IBuildLogStatus = 'doing' | 'success' | 'error' | 'info' | 'warn';
export interface IGlobalService {
    isDev(): any;
    setIsDev?: (func: FN) => void;
    getAppVersion(): Promise<string>;
    setGetAppVersion?: (func: FN) => void;
    showBuildLog2(statusMsgId: string, type: 'compile', status: IBuildLogStatus, text: string): Promise<void>;
    setShowBuildLog2?: (func: FN) => void;
}
export declare const IGlobalService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IGlobalService>;
