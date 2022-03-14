import type { HexString } from '../types';
interface Registry {
    get: (...params: unknown[]) => unknown;
}
interface Codec {
    readonly registry: Registry;
    toHex(isLe?: boolean): HexString;
    toU8a: (isBare?: unknown) => Uint8Array;
}
export declare function isCodec<T extends Codec = Codec>(value?: unknown): value is T;
export {};
