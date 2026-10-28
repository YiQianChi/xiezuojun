import { MiniProgramDevtools } from '../../../../../types';
interface ICompileData {
    projectPath: string;
    root: string;
    filePath: string;
    code: Buffer | String;
    setting: MiniProgramDevtools.IProjectSetting;
    babelRoot: string;
    babelIgnore: any[];
}
declare function compileJS(data: ICompileData): Promise<{
    error: null;
    isLargeFile: boolean;
    isBabelIgnore: boolean;
    map: string | undefined;
    code: string;
    helpers: never[];
} | {
    error: {
        path: string;
    } | {
        path: string;
        name: string;
        message: string;
        stack?: string;
        code?: number | string;
    } | {
        path: string;
        message: string;
        code: number;
    };
    isLargeFile?: undefined;
    isBabelIgnore?: undefined;
    map?: undefined;
    code?: undefined;
    helpers?: undefined;
} | {
    error: null;
    isLargeFile: false;
    isBabelIgnore: false;
    map: string;
    code: string;
    helpers: string[];
}>;
export = compileJS;
