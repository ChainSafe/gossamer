import type { Callback } from '@polkadot/types/types';
import type { UnsubscribePromise } from '../types';
export declare type CombinatorCallback<T extends unknown[]> = Callback<T>;
export interface CombinatorFunction {
    (cb: Callback<any>): UnsubscribePromise;
}
export declare class Combinator<T extends unknown[] = unknown[]> {
    #private;
    constructor(fns: (CombinatorFunction | [CombinatorFunction, ...unknown[]])[], callback: CombinatorCallback<T>);
    protected _allHasFired(): boolean;
    protected _createCallback(index: number): (value: any) => void;
    protected _triggerUpdate(): void;
    unsubscribe(): void;
}
