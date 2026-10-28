import { MiniProgramCore } from '~/types';
export interface IPluginFileReaderService {
    readFile(project: MiniProgramCore.IPreCompileProject, relativePath: string): string | Promise<string>;
    exists(project: MiniProgramCore.IPreCompileProject, basePath: string, fileName: string): boolean | Promise<boolean>;
    stat(project: MiniProgramCore.IPreCompileProject, relativePath: string): {
        mtimeMs: number;
    } | undefined | Promise<{
        mtimeMs: number;
    } | undefined>;
}
export declare const IPluginFileReaderService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IPluginFileReaderService>;
