export interface IGenTempFile {
    genPath(): Promise<string>;
}
export declare const IGenTempFile: import("../../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IGenTempFile>;
