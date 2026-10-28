export interface ICompileWxssOptions {
    files: string[];
    pageCount: number;
    cwd: string;
    lazyload?: boolean;
    subPackage?: string;
    replaceContent: Record<string, string>;
    isMiniAppProject?: boolean;
    contents?: string[];
}
export type ICompileWxssResult = Record<string, string>;
export interface ICompilewxssService {
    compileWxss: (options: ICompileWxssOptions) => Promise<ICompileWxssResult>;
}
export declare const ICompilewxssService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<ICompilewxssService>;
