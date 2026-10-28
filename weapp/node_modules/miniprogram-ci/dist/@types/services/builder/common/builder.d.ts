import { BaseCoreCompiler } from '../../..';
import type { PackOptionsHelper } from '../../..';
import type { MiniProgramDevtools } from '~/types';
export * from '../../..';
export interface IGetBuilderOptions {
    targetPlatform?: 'mini-ios' | 'mini-android' | 'mini-ohos';
    targetPlatformDefines?: {
        [key: string]: any;
    };
}
export interface IBuilder {
    getCompiler: (project: IProject, opts?: IGetBuilderOptions) => Promise<BaseCoreCompiler>;
    getBabelHelperVersion: () => string;
    getWorkletVersion: () => string;
    getHelperContent(setting: MiniProgramDevtools.IProjectSetting, mod: string): Promise<string>;
    getHelperName(setting: MiniProgramDevtools.IProjectSetting, mod: string): string;
    isValidHelperFunc(funcName: string): string | void;
    getHelperOutputPath(setting: MiniProgramDevtools.IProjectSetting): string;
    getHelperDeps(setting: MiniProgramDevtools.IProjectSetting, helpers: Set<string>): string[];
    getPackOptionsHelper(): PackOptionsHelper;
}
export declare const IBuilder: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IBuilder>;
