export interface IDirConfig {
    WeappCompileCache: string;
    WeappMiniApp: string;
}
export interface IAppDirService {
    init(dirConfig?: IDirConfig): Promise<IDirConfig>;
    get<T extends keyof IDirConfig>(dirType: T): Promise<IDirConfig[T]>;
    cleanDir(dirType: keyof IDirConfig): Promise<void>;
}
export declare const IAppDirService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IAppDirService>;
