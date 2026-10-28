import { IAppDirService, IDirConfig } from '../common/appDirService';
export declare class AppDirDevtoolsService implements IAppDirService {
    init(): Promise<IDirConfig>;
    get<T extends keyof IDirConfig>(dirType: T): Promise<IDirConfig[T]>;
    cleanDir(dirType: keyof IDirConfig): Promise<void>;
}
