import { ICompileWxssOptions, ICompileWxssResult, ICompilewxssService } from '../common/compilewxssService';
export declare class NWCompilewxssService implements ICompilewxssService {
    compileWxss(options: ICompileWxssOptions): Promise<ICompileWxssResult>;
}
