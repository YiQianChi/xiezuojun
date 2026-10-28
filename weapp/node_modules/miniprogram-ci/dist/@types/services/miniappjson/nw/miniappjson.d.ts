import { IMiniappJson } from '../common/miniappjson';
export declare class NWMiniappJson implements IMiniappJson {
    private _isMiniAppProject;
    setIsMiniAppProject(func: FN): void;
    isMiniAppProject(project?: IProject): boolean;
}
