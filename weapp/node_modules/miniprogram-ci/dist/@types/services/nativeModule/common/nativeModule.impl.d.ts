import { INativeModuleService } from './nativeModule';
export declare class nativeModuleImpl implements INativeModuleService {
    private _WXMLParsePath;
    private _WXSSParsePath;
    private _GlassEaselLazyModulePath;
    private _GlassEaselTemplateCompilerPath;
    setWXMLParsePath(path: string): Promise<void>;
    getWXMLParsePath(): Promise<string>;
    setWXSSParsePath(path: string): Promise<void>;
    getWXSSParsePath(): Promise<string>;
    setGlassEaselTemplateCompilerPath(path: string): Promise<void>;
    getGlassEaselTemplateCompilerPath(): Promise<string>;
}
