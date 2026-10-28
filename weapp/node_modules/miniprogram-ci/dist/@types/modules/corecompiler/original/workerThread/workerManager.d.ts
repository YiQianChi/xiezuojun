import { MiniProgramCore } from '../../../../types';
export interface IWorkerTaskRecord {
    workerId: number | string;
    taskName: string;
    duration: number;
}
export declare const AbortEvent = "WorkerTaskAborted";
declare class WorkerManager {
    private _taskQueue;
    private _workerPool;
    private _max_pool_size;
    private _taskRecords;
    constructor(size: number, isDevtools?: boolean);
    runTask(name: string, data: any, onStatusUpdate?: MiniProgramCore.FN<void>): Promise<unknown>;
    getTaskRecords(): IWorkerTaskRecord[];
    clearTaskRecords(): void;
    abort(name: string): void;
    private _run;
    private getFreeWorker;
    private allocWorker;
    private workerCount;
    private onWorkerExit;
    private _actualWorkerPoolSize;
    private _doneThreadSet;
    private onTaskDone;
}
export declare const getWorkerManager: (isDevtools?: boolean) => WorkerManager;
export {};
