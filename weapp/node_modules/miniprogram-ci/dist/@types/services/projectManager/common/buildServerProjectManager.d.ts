import { IBuilderFileServiceContext } from '~/services/fs/common/fileService';
export interface IBuildServerProjectManagerService {
    getProject: (ctx: IBuilderFileServiceContext) => Promise<IBackendProject>;
}
export declare const IProjectManagerService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IBuildServerProjectManagerService>;
