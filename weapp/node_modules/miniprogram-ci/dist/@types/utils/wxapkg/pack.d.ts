export interface IFile {
    path: string;
    data: Buffer;
}
export declare function packCore(files: IFile[]): Buffer<ArrayBuffer>;
export default function pack(file: string, distPath: string, cwd: string, ignores?: string[]): Promise<Buffer<ArrayBuffer>>;
