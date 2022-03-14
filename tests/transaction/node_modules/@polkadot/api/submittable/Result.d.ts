import type { DispatchError, DispatchInfo, EventRecord, ExtrinsicStatus, Hash } from '@polkadot/types/interfaces';
import type { AnyJson, ISubmittableResult } from '@polkadot/types/types';
import type { SubmittableResultValue } from './types';
export declare class SubmittableResult implements ISubmittableResult {
    readonly dispatchError?: DispatchError;
    readonly dispatchInfo?: DispatchInfo;
    readonly internalError?: Error;
    readonly events: EventRecord[];
    readonly status: ExtrinsicStatus;
    readonly txHash: Hash;
    readonly txIndex?: number;
    constructor({ dispatchError, dispatchInfo, events, internalError, status, txHash, txIndex }: SubmittableResultValue);
    get isCompleted(): boolean;
    get isError(): boolean;
    get isFinalized(): boolean;
    get isInBlock(): boolean;
    get isWarning(): boolean;
    /**
     * @description Filters EventRecords for the specified method & section (there could be multiple)
     */
    filterRecords(section: string, method: string | string[]): EventRecord[];
    /**
     * @description Finds an EventRecord for the specified method & section
     */
    findRecord(section: string, method: string | string[]): EventRecord | undefined;
    /**
     * @description Creates a human representation of the output
     */
    toHuman(isExtended?: boolean): AnyJson;
}
