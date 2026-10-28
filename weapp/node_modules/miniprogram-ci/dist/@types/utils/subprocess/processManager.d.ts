import { ISummerProcessDiagnosticProgress } from '../summerProcessLogger';
import { MiniProgramCore, MiniProgramDevtools } from '../../types';
export declare class SubProcessProxy {
    project: MiniProgramCore.IPreCompileProject;
    private entryPath;
    private passData;
    private initOptions;
    private inspectPort;
    private progressUpdate?;
    private ownerId;
    readonly diagnosticId: string;
    private process?;
    private taskMap;
    private taskId;
    forkId: number;
    private _checkReadyTask?;
    private _cpPromise?;
    private _forkProcessStartTime?;
    private _forkDoneTime?;
    private lastDiagnosticProgress?;
    onChildEvent?: (name: string, data: any) => void;
    constructor(project: MiniProgramCore.IPreCompileProject, entryPath: string, passData: {
        cachePath?: string;
        miniappDirPath?: string;
        devtoolsVersion?: string;
    }, initOptions: any, inspectPort?: number, progressUpdate?: MiniProgramDevtools.ProgressUpdate | undefined, ownerId?: string);
    ready(): Promise<any>;
    private sendProcessMessage;
    private init;
    destroy(): void;
    private getInProcessHandler;
    private forkProcess;
    private onMessage;
    getLastDiagnosticProgress(): ISummerProcessDiagnosticProgress | undefined;
    getDiagnosticContext(): string;
    private onResponse;
    sendEvent(name: string, data: any): Promise<void>;
    runTask<T = any>(name: string, data?: any, progressUpdate?: MiniProgramDevtools.ProgressUpdate): Promise<T>;
}
