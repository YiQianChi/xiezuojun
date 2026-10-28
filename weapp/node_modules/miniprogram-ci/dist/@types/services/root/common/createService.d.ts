import { ServiceIdentifier } from '../../common/vs/platform/instantiation/common/instantiation';
import { SyncDescriptor } from '../../common/vs/platform/instantiation/common/descriptors';
import { vsbase } from '../../common/vsbase';
type IInstantiationService = vsbase.vs.platform.instantiation.common.instantiation.$types.IInstantiationService;
type ServiceCollection = vsbase.vs.platform.instantiation.common.serviceCollection.$types.ServiceCollection;
declare const IInstantiationService: ServiceIdentifier<import("../../common/vs/platform/instantiation/common/instantiation").IInstantiationService>;
declare const ServiceCollection: typeof import("../../common/vs/platform/instantiation/common/serviceCollection").ServiceCollection;
export declare function createService(contributedServices: Array<[ServiceIdentifier<any>, SyncDescriptor<any>]>): {
    instantiationService: IInstantiationService;
    serviceCollection: ServiceCollection;
};
export {};
