export declare const weapp = "weapp";
export declare const game = "game";
export declare const plugin = "plugin";
export declare const gamePlugin = "gamePlugin";
export declare const isGameKindCompileType: (compileType?: string) => compileType is "game" | "gamePlugin";
export declare const isPluginKindCompileType: (compileType?: string) => compileType is "plugin" | "gamePlugin";
export declare const isMPKindCompileType: (compileType?: string) => compileType is "weapp" | "plugin";
export declare const isGameKindProject: (project: {
    compileType?: string;
}) => boolean;
export declare const isPluginKindProject: (project: {
    compileType?: string;
}) => boolean;
export declare const isMPKindProject: (project: {
    compileType?: string;
}) => boolean;
export declare const isValidCompileType: (compileType: string) => compileType is "weapp" | "game" | "plugin" | "gamePlugin";
