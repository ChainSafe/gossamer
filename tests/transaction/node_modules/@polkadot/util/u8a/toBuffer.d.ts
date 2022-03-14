/// <reference types="node" />
/**
 * @name u8aToBuffer
 * @summary Creates a Buffer object from a hex string.
 * @description
 * `null` inputs returns an empty `Buffer` result. `UInt8Array` input values return the actual bytes value converted to a `Buffer`. Anything that is not a `UInt8Array` throws an error.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aToBuffer } from '@polkadot/util';
 *
 * console.log('Buffer', u8aToBuffer('0x123480001f'));
 * ```
 */
export declare function u8aToBuffer(value?: Uint8Array | null): Buffer;
