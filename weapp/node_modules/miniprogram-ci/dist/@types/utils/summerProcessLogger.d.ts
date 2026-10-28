export interface ISummerProcessLogger {
    log: (tag: string, msg: string, duration?: number) => void;
    info: (tag: string, msg: string, duration?: number) => void;
    warn: (tag: string, msg: string, duration?: number) => void;
    error: (tag: string, msg: string, duration?: number) => void;
}
export interface ISummerProcessDiagnosticProgress {
    stage: string;
    processed: number;
    total: number;
}
export declare function setSummerProcessLogger(logger: ISummerProcessLogger): void;
export declare function getSummerProcessLogger(): ISummerProcessLogger;
export declare function setSummerProcessDiagnosticReporter(reporter: (progress: ISummerProcessDiagnosticProgress) => void): void;
export declare function reportSummerProcessDiagnosticProgress(progress: ISummerProcessDiagnosticProgress): void;
