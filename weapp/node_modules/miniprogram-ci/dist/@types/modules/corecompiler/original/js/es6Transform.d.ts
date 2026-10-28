import type { RawSourceMap } from 'source-map';
interface IOptions {
    code: string;
    filePath: string;
    inputSourceMap?: RawSourceMap;
}
declare const es6Compile: (options: IOptions) => {
    code: string | null | undefined;
    map: RawSourceMap;
    error?: undefined;
} | {
    error: {
        message: string;
        code: number;
    };
    code?: undefined;
    map?: undefined;
};
export = es6Compile;
