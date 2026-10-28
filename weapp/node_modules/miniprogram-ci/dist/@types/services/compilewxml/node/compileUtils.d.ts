import { ICompileWxmlOptions, ICompileResult, ICompileWxmlToGlassEasel, IGlassEaselCompileResult } from '../common/compilewxmlService';
export declare function compileUseWCC(options: Exclude<ICompileWxmlOptions, 'replaceContent'>): Promise<ICompileResult>;
export declare function compileWxmlToGlassEasel(options: ICompileWxmlToGlassEasel): Promise<IGlassEaselCompileResult>;
