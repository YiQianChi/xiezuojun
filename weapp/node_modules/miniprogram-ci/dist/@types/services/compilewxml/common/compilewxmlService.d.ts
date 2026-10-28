export interface IGlassEaselCompileResult {
    templateResults: Record<string, string>;
    wxmlDependencies: Record<string, string[]>;
    scriptResult: string;
    runtimeString: string;
    runtimeGlobal?: string;
    gdcContent?: string;
    wxsEnvContent: string;
}
export interface ICompileWxmlToGlassEasel {
    wxmlAndWxsFiles: Record<string, string>;
}
export interface ICompileWxmlOptions {
    files: string[];
    configContent: string;
    configSplit: string;
    isCut: boolean;
    cwd: string;
    genfuncName: string;
    hasWxs?: boolean;
    isPlugin?: boolean;
    lazyloadConfig?: string;
    replaceContent: Record<string, string>;
    isMiniAppProject?: boolean;
    debugWXS?: boolean;
    debug?: boolean;
}
export interface INormalResult {
    code: string;
    name: string;
}
export interface ILazyResult {
    pages: Record<string, string>;
    names: Record<string, string>;
}
export type ICompileResult = INormalResult | ILazyResult;
export interface ICompilewxmlService {
    compileWxml: (options: ICompileWxmlOptions) => Promise<ICompileResult>;
    compileWxmlToGlassEasel: (options: ICompileWxmlToGlassEasel) => Promise<IGlassEaselCompileResult>;
}
export declare const ICompilewxmlService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<ICompilewxmlService>;
