import { IPerformanceReportService } from '../common/performance';
export declare class NWPerformanceReportService implements IPerformanceReportService {
    private _reportPerf;
    private _getReportName;
    setReportPerf: (fn: FN) => void;
    setGetReportName: (fn: FN) => void;
    reportPerf(name: string, hrTime: number): Promise<any>;
    getReportName(): any;
}
