export interface INativeModuleService {
    setWXMLParsePath: (path: string) => Promise<void>;
    getWXMLParsePath: () => Promise<string>;
    setWXSSParsePath: (path: string) => Promise<void>;
    getWXSSParsePath: () => Promise<string>;
    setGlassEaselTemplateCompilerPath: (path: string) => Promise<void>;
    getGlassEaselTemplateCompilerPath: () => Promise<string>;
}
export declare const INativeModuleService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<INativeModuleService>;
