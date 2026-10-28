export interface IStaticConfigService {
    get(key: string): any;
    setGet?: (func: FN) => void;
}
export declare const IStaticConfigService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IStaticConfigService>;
