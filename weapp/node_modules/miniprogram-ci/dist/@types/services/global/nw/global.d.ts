import { IGlobalService, IBuildLogStatus } from '../common/global';
export declare class NWGlobalService implements IGlobalService {
    private _isDev;
    private _getAppVersion;
    private _showBuildLog2;
    setShowBuildLog2(func: FN): void;
    setIsDev(func: FN): void;
    setGetAppVersion(func: FN): void;
    isDev(): Promise<any>;
    getAppVersion(): Promise<any>;
    showBuildLog2(statusMsgId: string, type: 'compile', status: IBuildLogStatus, text?: string): Promise<any>;
}
