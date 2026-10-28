import { MiniProgramCore } from '~/types';
export interface IMultiTaskManagerService {
    runTask(taskName: string, data: any, onStatusUpdate: MiniProgramCore.FN<void>): Promise<any>;
}
export declare const IMultiTaskManagerService: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IMultiTaskManagerService>;
