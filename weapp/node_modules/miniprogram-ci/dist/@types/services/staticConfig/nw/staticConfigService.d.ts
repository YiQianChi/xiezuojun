import { IStaticConfigService } from '../common/staticConfigService';
export declare class NWStaticConfigService implements IStaticConfigService {
    private _get;
    setGet(func: FN): void;
    get(key: string): any;
}
