export interface ILocalesService {
    getLocale(): any;
    setLocale?: (func: FN) => void;
    getLocales: () => Record<string, any>;
    setGetLocales?: (func: FN) => void;
}
export declare const ILocalesService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<ILocalesService>;
