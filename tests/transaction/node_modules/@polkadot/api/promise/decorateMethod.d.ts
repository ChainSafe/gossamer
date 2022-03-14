import type { Observable } from 'rxjs';
import type { Codec } from '@polkadot/types/types';
import type { DecorateFn, DecorateMethodOptions, ObsInnerType, StorageEntryPromiseOverloads } from '../types';
interface Tracker<T> {
    reject: (value: Error) => Observable<never>;
    resolve: (value: T) => void;
}
declare type CodecReturnType<T extends (...args: unknown[]) => Observable<Codec>> = T extends (...args: any) => infer R ? R extends Observable<Codec> ? ObsInnerType<R> : never : never;
export declare function promiseTracker<T>(resolve: (value: T) => void, reject: (value: Error) => void): Tracker<T>;
/**
 * @description Decorate method for ApiPromise, where the results are converted to the Promise equivalent
 */
export declare function toPromiseMethod<M extends DecorateFn<CodecReturnType<M>>>(method: M, options?: DecorateMethodOptions): StorageEntryPromiseOverloads;
export {};
