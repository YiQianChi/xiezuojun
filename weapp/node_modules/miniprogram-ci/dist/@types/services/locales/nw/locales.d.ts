import { ILocalesService } from '../common/locales';
export declare class NWLocalesService implements ILocalesService {
    private _getLocale;
    private _getLocales;
    setGetLocales(func: FN): void;
    setLocale(func: FN): void;
    getLocale(): Promise<any>;
    getLocales(): any;
}
