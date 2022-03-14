/**
 * @name randomAsU8a
 * @summary Creates a Uint8Array filled with random bytes.
 * @description
 * Returns a `Uint8Array` with the specified (optional) length filled with random bytes.
 * @example
 * <BR>
 *
 * ```javascript
 * import { randomAsU8a } from '@polkadot/util-crypto';
 *
 * randomAsU8a(); // => Uint8Array([...])
 * ```
 */
export declare function randomAsU8a(length?: number): Uint8Array;
/**
 * @name randomAsHex
 * @description Creates a hex string filled with random bytes.
 */
export declare const randomAsHex: (length?: number | undefined) => `0x${string}`;
