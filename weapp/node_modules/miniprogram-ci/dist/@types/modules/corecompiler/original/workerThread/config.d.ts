export declare enum ETaskStatus {
    waiting = 0,
    progress = 1,
    done = 2
}
export declare const TASK_NAME: {
    COMPILE_JS: string;
    COMPILE_WXSS: string;
    MINIFY_WXML: string;
    SUMMER_HOOK: string;
    PIPE_TASK: string;
};
export declare const COMMAND: {
    INIT_START: string;
    INIT_READY: string;
    RUN_TASK: string;
    TASK_DONE: string;
    CALL_FUNC: string;
    CALL_FUNC_RESULT: string;
    CHILD_PROCESS_READY: string;
    SEND_LOG: string;
};
export declare enum EWorkerStatus {
    initiating = 0,
    free = 1,
    busy = 2,
    dying = 3
}
export declare enum EChildProcessStatus {
    free = 0,
    busy = 1,
    fullload = 2,
    dying = 3
}
export declare const SUICIDE_TIME: {
    devtools: number;
    'miniprogram-ci': number;
    webide: number;
};
