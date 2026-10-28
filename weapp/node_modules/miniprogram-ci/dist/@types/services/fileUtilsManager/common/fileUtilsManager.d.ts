export interface IFileUtilsManagerService {
    projectFileUtils(projectPath: string, options: Record<string, any>): Promise<any>;
    setProjectFileUtils?: (func: FN) => void;
}
export declare const IFileUtilsManagerService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IFileUtilsManagerService>;
