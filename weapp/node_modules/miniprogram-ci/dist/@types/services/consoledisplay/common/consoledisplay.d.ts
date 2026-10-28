export interface IConsoleDisplayService {
    display: (arg: Record<string, any>) => void;
    setDisplay?: (func: FN) => void;
}
export declare const IConsoleDisplayService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IConsoleDisplayService>;
