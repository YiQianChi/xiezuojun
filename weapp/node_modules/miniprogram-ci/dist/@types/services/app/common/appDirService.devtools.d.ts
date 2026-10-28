import { IAppDirService, IDirConfig } from './appDirService';
export declare class AppDirDevtoolsService implements IAppDirService {
    private _config;
    init(config: IDirConfig): Promise<IDirConfig>;
    get<T extends keyof IDirConfig>(dirType: T): Promise<IDirConfig[T]>;
    cleanDir(dirType: keyof IDirConfig): Promise<void>;
}
