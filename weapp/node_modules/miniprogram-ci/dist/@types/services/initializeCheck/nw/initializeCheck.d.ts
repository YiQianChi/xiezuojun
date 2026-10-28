import { EInitializeSteps, EInitializeStepsStatus, IInitializeCheck } from '../common/initializeCheck';
export declare class InitializeCheck implements IInitializeCheck {
    notifiedStatus(step: EInitializeSteps, status: EInitializeStepsStatus): void;
}
