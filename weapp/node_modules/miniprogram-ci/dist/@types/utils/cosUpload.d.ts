export type IRegion = 'ap-shanghai';
export type ISignOptsHeaders = {
    [key: string]: string;
} | Array<{
    key: string;
    value: string;
}>;
export declare function getCosEndpointOptions(): {
    Domain?: string;
    ServiceDomain?: string;
};
export declare function getSignOptsHeaders(bucket: string, region: string): ISignOptsHeaders;
export interface IUploadToCosCoreParam {
    uploadBuf: Buffer | string;
    region: IRegion;
    getUploadInfoFunc: FGetUploadInfoFunc;
    getAuthFunc: FGetAuthFunc;
    onProgress?: (progressData: any) => void;
}
export interface IUploadToCosCoreRes {
    err?: string;
    data?: string;
}
export interface IUploadInfo {
    expired_time: string;
    start_time: string;
    object: string;
    bucket: string;
    crypt_key: string;
    task_id: string;
    checksum: string;
}
export interface IUploadReq extends IUploadInfo {
    action: string;
    uploadId?: string;
}
export interface IAuthResData {
    Authorization: string;
    SecurityToken: string;
    AuthExpireTime: number;
}
export interface IReqCache extends IUploadReq, IAuthResData {
}
type FGetUploadInfoFunc = () => Promise<IUploadInfo>;
type FGetAuthFunc = (data: IUploadReq) => Promise<IAuthResData>;
export declare function uploadToCosCore({ uploadBuf, region, getUploadInfoFunc, getAuthFunc, onProgress, }: IUploadToCosCoreParam): Promise<IUploadToCosCoreRes>;
export declare function uploadToCosInCI(): void;
export {};
