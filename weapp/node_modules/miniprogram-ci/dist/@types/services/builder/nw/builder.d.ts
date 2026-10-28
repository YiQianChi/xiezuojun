import { BaseCoreCompiler } from '../../..';
import type { PackOptionsHelper } from '../../..';
import type { MiniProgramDevtools } from '~/types';
import { IBuilder } from '../common/builder';
export declare class NWBuilder implements IBuilder {
    getCompiler(): Promise<BaseCoreCompiler>;
    getBabelHelperVersion(): string;
    getWorkletVersion(): any;
    getHelperContent(setting: MiniProgramDevtools.IProjectSetting, mod: string): Promise<string>;
    getHelperName(setting: MiniProgramDevtools.IProjectSetting, mod: string): string;
    isValidHelperFunc(funcName: string): "babel" | "swc" | undefined;
    getHelperOutputPath(setting: MiniProgramDevtools.IProjectSetting): string;
    getHelperDeps(setting: MiniProgramDevtools.IProjectSetting, helpers: Set<string>): string[];
    getPackOptionsHelper(): PackOptionsHelper;
}
