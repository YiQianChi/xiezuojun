import { MiniProgramCore } from "~/types";
import { IMultiTaskManagerService } from "../common/multiTaskManager";
export declare class WebideMultiTaskManagerService implements IMultiTaskManagerService {
    runTask(taskName: string, data: any, onStatusUpdate: MiniProgramCore.FN<void>): Promise<any>;
}
