declare type BitLength = 256 | 512;
/**
 * @name hmacShaAsU8a
 * @description creates a Hmac Sha (256/512) Uint8Array from the key & data
 */
export declare function hmacShaAsU8a(key: Uint8Array | string, data: Uint8Array, bitLength?: BitLength, onlyJs?: boolean): Uint8Array;
/**
 * @name hmacSha256AsU8a
 * @description creates a Hmac Sha256 Uint8Array from the key & data
 */
export declare const hmacSha256AsU8a: (key: Uint8Array | string, data: Uint8Array, onlyJs?: boolean | undefined) => Uint8Array;
/**
 * @name hmacSha512AsU8a
 * @description creates a Hmac Sha512 Uint8Array from the key & data
 */
export declare const hmacSha512AsU8a: (key: Uint8Array | string, data: Uint8Array, onlyJs?: boolean | undefined) => Uint8Array;
export {};
