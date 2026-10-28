import { vsbase } from './vsbase';
declare const SyncDescriptor: typeof import("./vs/platform/instantiation/common/descriptors").SyncDescriptor;
type ServiceIdentifier<T> = vsbase.vs.platform.instantiation.common.instantiation.$types.ServiceIdentifier<T>;
type SyncDescriptor<T> = vsbase.vs.platform.instantiation.common.descriptors.$types.SyncDescriptor<T>;
type IConstructorSignature0<T> = vsbase.vs.platform.instantiation.common.descriptors.$types.IConstructorSignature0<T>;
export declare function registerSingleton<T>(id: ServiceIdentifier<T>, ctor: IConstructorSignature0<T>, supportsDelayedInstantiation?: boolean, autoRegisterInRootServiceCollection?: boolean): void;
export declare function getSingletonServiceDescriptors(): Array<[ServiceIdentifier<any>, SyncDescriptor<any>]>;
export {};
