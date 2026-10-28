import { IProjectManagerService } from '../common/projectManager';
export declare class NWProjectManagerService implements IProjectManagerService {
    private _getAppIdAndAttr;
    private _getCurrentRuntimeConfig;
    setGetCurrentRuntimeConfig(func: FN): void;
    setGetAppIdAndAttr(func: FN): void;
    getAppIdAndAttr(): Promise<any>;
    getCurrentRuntimeConfig(): any;
}
