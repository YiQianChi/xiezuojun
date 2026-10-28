import { ICompileWxssOptions, ICompileWxssResult, ICompilewxssService } from '../common/compilewxssService';
export declare class ElectronCompilewxssService implements ICompilewxssService {
    compileWxss(options: ICompileWxssOptions): Promise<ICompileWxssResult>;
}
