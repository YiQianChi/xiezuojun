import { IGenTempFile } from './genTempFileService';
export declare class devtoolsGenTempFile implements IGenTempFile {
    genPath(): Promise<string>;
}
