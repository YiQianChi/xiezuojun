import { ICompileWxmlOptions, ICompileResult, ICompilewxmlService, ICompileWxmlToGlassEasel, IGlassEaselCompileResult } from '../common/compilewxmlService';
export declare class WebideCompilewxmlService implements ICompilewxmlService {
    compileWxml(options: ICompileWxmlOptions): Promise<ICompileResult>;
    compileWxmlToGlassEasel(options: ICompileWxmlToGlassEasel): Promise<IGlassEaselCompileResult>;
}
