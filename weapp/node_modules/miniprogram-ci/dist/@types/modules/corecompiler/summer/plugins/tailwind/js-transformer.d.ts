export interface TransformJsOptions {
    classContextOnly?: boolean;
}
export declare function transformJs(code: string, classSet: Set<string> | null, options?: TransformJsOptions): string;
