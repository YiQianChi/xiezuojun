import { IFileUtilsManagerService } from '../common/fileUtilsManager';
export declare class NWFileUtilsManagerService implements IFileUtilsManagerService {
    private _projectFileUtils;
    setProjectFileUtils(func: FN): void;
    projectFileUtils(projectPath: string, options: Record<string, any>): Promise<any>;
}
