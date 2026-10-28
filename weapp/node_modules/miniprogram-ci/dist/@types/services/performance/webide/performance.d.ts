import { IPerformanceReportService } from '../common/performance';
export declare class WebidePerformanceReportService implements IPerformanceReportService {
    reportPerf(name: string, hrTime: number): Promise<void>;
    getReportName(): {
        [key: string]: string;
    };
}
