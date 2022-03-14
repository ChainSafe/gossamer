/**
 * @name base64Validate
 * @summary Validates a base64 value.
 * @description
 * Validates that the supplied value is valid base64
 */
export declare const base64Validate: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
/**
 * @name isBase64
 * @description Checks if the input is in base64, returning true/false
 */
export declare const isBase64: (value?: unknown, ipfsCompat?: boolean | undefined) => value is string;
/**
 * @name base64Decode
 * @summary Decodes a base64 value.
 * @description
 * From the provided input, decode the base64 and return the result as an `Uint8Array`.
 */
export declare const base64Decode: (value: string, ipfsCompat?: boolean | undefined) => Uint8Array;
/**
 * @name base64Encode
 * @summary Creates a base64 value.
 * @description
 * From the provided input, create the base64 and return the result as a string.
 */
export declare const base64Encode: (value: import("@polkadot/util/types").U8aLike, ipfsCompat?: boolean | undefined) => string;
