import { MiniProgramDevtools } from '../../../types';
import { Analyzer } from '../../../common/code-analyse';
export interface IdleCompileHost {
    analyzer: Analyzer | null;
    project: {
        miniprogramRoot: string;
    };
    devCodeCacheByPath: Map<string, {
        isValid: boolean;
        codeFile: MiniProgramDevtools.CodeFile | MiniProgramDevtools.CodeError;
    }>;
    idleGetCodeByFileList(graphId: MiniProgramDevtools.GraphId, fileList: string[]): Promise<MiniProgramDevtools.CodeFiles>;
    log(type: 'info' | 'warn', message: string): void;
}
export declare class IdleCompileScheduler {
    private host;
    private graphId;
    private list;
    private launchPath;
    private timer;
    private activeTaskCount;
    private destroyed;
    private running;
    constructor(host: IdleCompileHost, graphId: MiniProgramDevtools.GraphId);
    notifyTaskStart(): void;
    notifyTaskEnd(): void;
    updateLaunchPath(launchPath: string): void;
    stop(): void;
    updateGraphId(graphId: MiniProgramDevtools.GraphId): void;
    private schedule;
    private run;
    private genList;
}
