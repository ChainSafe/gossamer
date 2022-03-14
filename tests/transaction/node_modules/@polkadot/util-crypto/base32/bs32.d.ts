/**
 * @name base32Validate
 * @summary Validates a base32 value.
 * @description
 * Validates that the supplied value is valid base32, throwing exceptions if not
 */
export declare const base32Validate: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
/**
* @name isBase32
* @description Checks if the input is in base32, returning true/false
*/
export declare const isBase32: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
/**
 * @name base32Decode
 * @summary Delookup a base32 value.
 * @description
 * From the provided input, decode the base32 and return the result as an `Uint8Array`.
 */
export declare const base32Decode: (value: string, ipfsCompat?: boolean | undefined) => Uint8Array;
/**
* @name base32Encode
* @summary Creates a base32 value.
* @description
* From the provided input, create the base32 and return the result as a string.
*/
export declare const base32Encode: (value: import("@polkadot/util/types").U8aLike, ipfsCompat?: boolean | undefined) => string;
