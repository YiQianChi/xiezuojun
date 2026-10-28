import { IConsoleDisplayService } from '../common/consoledisplay';
export declare class NWConsoleDisplayService implements IConsoleDisplayService {
    private _display?;
    setDisplay(func: FN): void;
    display(arg: Record<string, any>): void | undefined;
}
