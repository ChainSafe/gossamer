/// <reference types="node" />
/**
 * @name shaAsU8a
 * @summary Creates a sha Uint8Array from the input.
 */
export declare const shaAsU8a: (value: string | Uint8Array | Buffer, bitLength?: 256 | 512 | undefined, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name sha256AsU8a
 * @summary Creates a sha256 Uint8Array from the input.
 */
export declare const sha256AsU8a: (data: string | Uint8Array | Buffer, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name sha512AsU8a
 * @summary Creates a sha512 Uint8Array from the input.
 */
export declare const sha512AsU8a: (data: string | Uint8Array | Buffer, onlyJs?: boolean | undefined) => Uint8Array;
