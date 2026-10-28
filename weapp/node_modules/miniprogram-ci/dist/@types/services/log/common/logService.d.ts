export interface ILogService {
    error: (...args: any) => void;
    info: (...args: any) => void;
    warn: (...args: any) => void;
    log: (...args: any) => void;
    debug: (...args: any) => void;
    context: (ctx: string, consoleObj?: Omit<ILogService, 'context'>) => ILogService;
}
export declare const ILogService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<ILogService>;
