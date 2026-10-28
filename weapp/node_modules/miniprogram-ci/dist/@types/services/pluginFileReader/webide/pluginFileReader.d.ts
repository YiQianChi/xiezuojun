import { MiniProgramCore } from '~/types';
import { IPluginFileReaderService } from '~/services/pluginFileReader/common/pluginFileReader';
export declare class WebIDEPluginFileReaderService implements IPluginFileReaderService {
    readFile(project: MiniProgramCore.IPreCompileProject, relativePath: string): string;
    exists(project: MiniProgramCore.IPreCompileProject, basePath: string, fileName: string): boolean;
    stat(project: MiniProgramCore.IPreCompileProject, relativePath: string): {
        mtimeMs: number;
    } | undefined;
}
