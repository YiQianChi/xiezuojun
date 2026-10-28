import { ICompileWxssOptions, ICompileWxssResult, ICompilewxssService } from '../common/compilewxssService';
export declare class WebideCompilewxssService implements ICompilewxssService {
    compileWxss(options: ICompileWxssOptions): Promise<ICompileWxssResult>;
}
