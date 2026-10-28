import { IProject, MiniProgramCI } from '../types';
export declare function transCompileType(project: IProject): "miniprogram" | "game" | "plugin" | "gamePlugin";
export declare function checkCodeQuality(project: IProject): Promise<MiniProgramCI.ICheckResultItem[]>;
