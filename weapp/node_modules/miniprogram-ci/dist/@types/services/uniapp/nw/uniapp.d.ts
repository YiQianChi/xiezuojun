import { IUniApp } from '../common/uniapp';
import { IStoreService } from '~/services/store/common/storeService';
export declare class NWUniApp implements IUniApp {
    protected readonly storeService: IStoreService;
    constructor(storeService: IStoreService);
    private _copyMiniAppJson;
    setCopyMiniAppJson(func: FN): void;
    copyMiniAppJson(project: IProject): boolean;
}
