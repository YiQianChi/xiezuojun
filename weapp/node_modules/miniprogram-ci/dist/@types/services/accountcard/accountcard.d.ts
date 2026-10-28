export interface IAccountCardResources {
    js: string;
    wxss: string;
    json: string;
    wxml: string;
    prependjs?: string;
}
export type ICardType = {
    type: number;
    displayName: string;
};
export interface IAccountCardService {
    init: () => Promise<void>;
    refresh: () => void;
    getAvailableAccountCardTypes: () => ICardType[];
    getAccountCardResources: (type: number) => IAccountCardResources;
}
export declare const IAccountCardService: import("../common/vs/platform/instantiation/common/instantiation").ServiceIdentifier<IAccountCardService>;
