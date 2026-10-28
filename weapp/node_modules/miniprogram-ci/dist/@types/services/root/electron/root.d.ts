import { IInstantiationService, ServiceIdentifier } from '../../common/vs/platform/instantiation/common/instantiation';
export declare class Root {
    private static _instance;
    protected instantiationService: IInstantiationService;
    constructor(instantiationService?: IInstantiationService);
    static get instance(): Root;
    static set instance(rootins: Root);
    static get iis(): IInstantiationService;
    static initIisWithServices(i: IInstantiationService): Root;
    static getService<T>(a: ServiceIdentifier<T>): Promise<T>;
}
