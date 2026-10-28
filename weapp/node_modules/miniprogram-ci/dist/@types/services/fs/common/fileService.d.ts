import { Stats } from 'fs';
export type IReadFileOptions = {
    encoding?: null | undefined | string;
    flag?: string;
} | null | string;
export interface IBuilderFileServiceContext {
    projectId: string;
    versionId: string;
    ticket: string;
}
export type IFileList = {
    name: string;
    key: string;
    lastModifiedTime: number;
    size: number;
    createTime: number;
    content?: string;
}[];
export interface IFileService {
    readFile: (filePath: string, options: IReadFileOptions, ctx: IBuilderFileServiceContext) => Promise<Buffer | string>;
    readFileList: (filePath: string[], ctx: IBuilderFileServiceContext) => Promise<{
        [filePath: string]: string;
    }>;
    exists: (filePath: string, ctx: IBuilderFileServiceContext) => Promise<boolean>;
    lstat: (filePath: string, ctx: IBuilderFileServiceContext) => Promise<Stats>;
    fileList: (ctx: IBuilderFileServiceContext) => Promise<IFileList>;
}
export declare const IFileService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IFileService>;
