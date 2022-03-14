import type { U8aLike } from '../types';
export declare const U8A_WRAP_ETHEREUM: Uint8Array;
export declare const U8A_WRAP_PREFIX: Uint8Array;
export declare const U8A_WRAP_POSTFIX: Uint8Array;
export declare function u8aIsWrapped(u8a: Uint8Array, withEthereum: boolean): boolean;
export declare function u8aUnwrapBytes(bytes: U8aLike): Uint8Array;
export declare function u8aWrapBytes(bytes: U8aLike): Uint8Array;
