import { MiniProgramCore } from '../types';
export declare class TaskStatus implements MiniProgramCore.ITaskStatus {
    private _id;
    protected _status: 'doing' | 'success';
    protected _msg: string;
    constructor(msg: string);
    get id(): string;
    get status(): "doing" | "success";
    get message(): string;
    done(): void;
    toString(): string;
}
