/**
 * @name base58Validate
 * @summary Validates a base58 value.
 * @description
 * Validates that the supplied value is valid base58, throwing exceptions if not
 */
export declare const base58Validate: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
/**
 * @name base58Decode
 * @summary Decodes a base58 value.
 * @description
 * From the provided input, decode the base58 and return the result as an `Uint8Array`.
 */
export declare const base58Decode: (value: string, ipfsCompat?: boolean | undefined) => Uint8Array;
/**
* @name base58Encode
* @summary Creates a base58 value.
* @description
* From the provided input, create the base58 and return the result as a string.
*/
export declare const base58Encode: (value: import("@polkadot/util/types").U8aLike, ipfsCompat?: boolean | undefined) => string;
/**
* @name isBase58
* @description Checks if the input is in base58, returning true/false
*/
export declare const isBase58: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
