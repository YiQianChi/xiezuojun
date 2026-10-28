export declare class CodeError extends Error {
    code: number;
    constructor(msg: string, code: number);
}
export type ErrorStage = 'builder' | 'backend';
export declare class CiError extends CodeError {
    errorStage: ErrorStage;
    errCode?: number;
    errMsg: string;
    constructor(errorStage: ErrorStage, errMsg: string, errCode?: number, code?: number);
}
export declare function parseBackendErrCode(errMsg?: string): number | undefined;
