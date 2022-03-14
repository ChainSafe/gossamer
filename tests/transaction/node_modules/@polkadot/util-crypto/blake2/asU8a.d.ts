import type { HexString } from '@polkadot/util/types';
declare type BitLength = 64 | 128 | 256 | 384 | 512;
/**
 * @name blake2AsU8a
 * @summary Creates a blake2b u8a from the input.
 * @description
 * From a `Uint8Array` input, create the blake2b and return the result as a u8a with the specified `bitLength`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { blake2AsU8a } from '@polkadot/util-crypto';
 *
 * blake2AsU8a('abc'); // => [0xba, 0x80, 0xa5, 0x3f, 0x98, 0x1c, 0x4d, 0x0d]
 * ```
 */
export declare function blake2AsU8a(data: HexString | Uint8Array | string, bitLength?: BitLength, key?: Uint8Array | null, onlyJs?: boolean): Uint8Array;
/**
 * @name blake2AsHex
 * @description Creates a blake2b hex from the input.
 */
export declare const blake2AsHex: (data: string | Uint8Array, bitLength?: BitLength | undefined, key?: Uint8Array | null | undefined, onlyJs?: boolean | undefined) => `0x${string}`;
export {};
