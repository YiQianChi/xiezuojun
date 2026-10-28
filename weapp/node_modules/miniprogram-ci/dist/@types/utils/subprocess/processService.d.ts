import type { MiniProgramDevtools } from '~/types';
export interface ProcessService {
    send: (msg: any) => void;
    kill: (signal: any) => boolean;
}
export declare const PROCESS_READY_MAX_TIMEOUT: number;
export declare function setForkReadyTimeoutHandler(fn: ((isFinal?: boolean) => Promise<boolean>) | undefined): void;
export declare function setForkReadySuccessHandler(fn: (() => void | Promise<void>) | undefined): void;
export declare function forkProcess(that: any, entryPath: string, projectPath: string, initMsg: MiniProgramDevtools.InitMsg, inspectPort?: number): Promise<ProcessService>;
