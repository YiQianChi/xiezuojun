import { AppJSON, IProject, MiniProgramCore } from '../types';
import crypto = require('crypto');
export interface IPageInfo {
    path: string;
    root: string;
    name?: string;
}
export declare const getAllPages: (appJSON: AppJSON.IAppJSON) => string[];
export declare const getAllPagesInfo: (appJSON: MiniProgramCore.IAnyObject) => IPageInfo[];
export declare function throwError(options: {
    code?: number;
    msg: string;
    filePath: string;
}): void;
export declare function checkUTF8(code: Buffer, filePath: string): string;
export declare enum ECheckPathType {
    NORMAL = "NORMAL",
    TAB_BAR_ICON = "TAB_BAR_ICON"
}
export declare function checkPath(options: {
    value: string;
    tips: string;
    filePath?: string;
    code?: number;
    checkPathType?: string;
}): void;
export declare function getAllTargetTypeFilesWithOtherTypeFilesOfSameName(project: IProject, targetType: string, otherTypes: string[], filter: string): string[];
export declare const workletVersion = "0.0.7";
export declare function generateMD5(string: crypto.BinaryLike): string;
export declare function random(): string;
