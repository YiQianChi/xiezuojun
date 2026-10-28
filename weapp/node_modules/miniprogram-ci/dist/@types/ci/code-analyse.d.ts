import { IProject } from '../types';
export declare function transCompileType(project: IProject): "miniprogram" | "game" | "plugin" | "gamePlugin";
export declare function analyseCode(project: IProject, option?: {
    silent: boolean;
}): Promise<import("../common/code-analyse").IAnalyseResult | undefined>;
