export interface IPerformanceReportService {
    setReportPerf?: (fn: FN) => void;
    setGetReportName?: (fn: FN) => void;
    reportPerf: (name: string, hrTime: number) => Promise<void>;
    getReportName(): {
        [key: string]: string;
    };
}
export declare const IPerformanceReportService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IPerformanceReportService>;
