import { Event } from 'vs/base/common/event';
export type GetValueType<T, Path extends unknown[]> = Path extends [infer K1, ...infer Rest] ? K1 extends keyof T ? GetValueType<T[K1], Rest> : undefined : T;
type ThunkAction = (dispatch: FN<void>, getState: FN) => Promise<any>;
export interface IStoreService {
    createStore: () => any;
    getState: () => any;
    subscribe: (fn: FN<void>) => FN<void>;
    unsubscribe: (fn: FN<void>) => FN<void>;
    getValue: <Path extends string[]>(...args: Path) => Promise<(GetValueType<any, Path>)>;
    getValueByRuntimeId: <Path extends string[]>(runtimeId: string, ...args: Path) => Promise<(GetValueType<any, Path>)>;
    onStateChange: Event<any>;
    dispatch(action: any | ThunkAction): any;
    setDispatchEanInfoSetConfirmInfo?: (func: FN) => void;
    dispatchEanInfoSetConfirmInfo?: (params: Record<string, any>) => void;
}
export declare const IStoreService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IStoreService>;
export {};
