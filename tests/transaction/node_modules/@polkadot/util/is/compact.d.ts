/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
interface Compact<T> {
    toBigInt(): bigint;
    toBn(): BN;
    toNumber(): number;
    unwrap(): T;
}
/**
 * @name isCompact
 * @summary Tests for SCALE-Compact-like object instance.
 */
export declare function isCompact<T>(value?: unknown): value is Compact<T>;
export {};
