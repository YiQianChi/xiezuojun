import { IGenTempFile } from './genTempFileService';
export declare class nodeGenTempFile implements IGenTempFile {
    genPath(): Promise<string>;
}
