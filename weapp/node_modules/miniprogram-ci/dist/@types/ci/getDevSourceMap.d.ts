import { IProject } from '../types';
export interface ISourceMapProgress {
    totalSize: number;
    downloadedSize: number;
    percentage: number;
}
export interface IGetDevSourceMapOption {
    project: IProject;
    robot: number;
    sourceMapSavePath: string;
    streaming?: boolean;
    sleep?: number;
    onProgress?: (info: ISourceMapProgress) => void;
}
export declare function getDevSourceMap(options: IGetDevSourceMapOption): Promise<any>;
