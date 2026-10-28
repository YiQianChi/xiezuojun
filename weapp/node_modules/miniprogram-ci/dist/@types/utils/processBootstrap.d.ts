interface IProcessBootstrapOptions {
    baseDir: string;
    hackRequirePath?: string;
    moduleFallbackPath?: string;
}
export declare function installProcessBootstrap({ baseDir, hackRequirePath, moduleFallbackPath, }: IProcessBootstrapOptions): void;
export {};
