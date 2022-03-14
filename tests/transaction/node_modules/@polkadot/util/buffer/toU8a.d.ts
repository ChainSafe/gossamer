/// <reference types="node" />
/**
 * @name bufferToU8a
 * @summary Creates a Uint8Array value from a Buffer object.
 * @description
 * `null` inputs returns an empty result, `Buffer` values return the actual value as a `Uint8Array`. Anything that is not a `Buffer` object throws an error.
 * @example
 * <BR>
 *
 * ```javascript
 * import { bufferToU8a } from '@polkadot/util';
 *
 * bufferToU8a(Buffer.from([1, 2, 3]));
 * ```
 */
export declare function bufferToU8a(buffer?: Buffer | number[] | null): Uint8Array;
