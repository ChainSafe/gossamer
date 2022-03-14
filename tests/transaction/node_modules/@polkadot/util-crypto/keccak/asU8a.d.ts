/// <reference types="node" />
/**
 * @name keccakAsU8a
 * @summary Creates a keccak Uint8Array from the input.
 * @description
 * From either a `string` or a `Buffer` input, create the keccak and return the result as a `Uint8Array`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { keccakAsU8a } from '@polkadot/util-crypto';
 *
 * keccakAsU8a('123'); // => Uint8Array
 * ```
 */
export declare const keccakAsU8a: (value: string | Uint8Array | Buffer, bitLength?: 256 | 512 | undefined, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name keccak256AsU8a
 * @description Creates a keccak256 Uint8Array from the input.
 */
export declare const keccak256AsU8a: (data: string | Uint8Array | Buffer, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name keccak512AsU8a
 * @description Creates a keccak512 Uint8Array from the input.
 */
export declare const keccak512AsU8a: (data: string | Uint8Array | Buffer, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name keccakAsHex
 * @description Creates a keccak hex string from the input.
 */
export declare const keccakAsHex: (value: string | Uint8Array | Buffer, bitLength?: 256 | 512 | undefined, onlyJs?: boolean | undefined) => `0x${string}`;
