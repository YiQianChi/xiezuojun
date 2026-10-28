import { vsbase } from '../common/vsbase';
export declare enum EServiceEnviroment {
    ELECTRON = "ELECTRON",
    SERVER = "SERVER",
    NW = "NW",
    CI = "CI",
    WEBIDE = "WEBIDE"
}
type ServiceIdentifier<T> = vsbase.vs.platform.instantiation.common.instantiation.$types.ServiceIdentifier<T>;
interface IDevtoolsServiceOptions {
    NativeModule: {
        WXMLParsePath: string;
        WXSSParsePath: string;
        GlassEaselLazyModulePath: string;
    };
    AppDirService: {
        WeappCompileCache: string;
    };
}
interface INodeServiceOptions {
}
export declare function initServices(env: EServiceEnviroment.CI | EServiceEnviroment.SERVER, serviceOptions: INodeServiceOptions, getVsBaseBundleFunc?: () => any): void;
export declare function initServices(env: EServiceEnviroment.NW | EServiceEnviroment.ELECTRON, serviceOptions: IDevtoolsServiceOptions, getVsBaseBundleFunc?: () => any): void;
export declare function getServicesEnv(): EServiceEnviroment.CI;
export declare function initServicesByEnv(): void;
declare const _default: <T>(id: ServiceIdentifier<T>) => T;
export default _default;
