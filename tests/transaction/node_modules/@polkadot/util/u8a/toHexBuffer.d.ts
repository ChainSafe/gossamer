import type { HexString } from '../types';
/**
 * @name u8aToHex
 * @summary Creates a hex string from a Uint8Array object.
 * @description
 * `UInt8Array` input values return the actual hex string. `null` or `undefined` values returns an `0x` string.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aToHex } from '@polkadot/util';
 *
 * u8aToHex(new Uint8Array([0x68, 0x65, 0x6c, 0x6c, 0xf])); // 0x68656c0f
 * ```
 */
export declare function u8aToHex(value?: Uint8Array | null, bitLength?: number, isPrefixed?: boolean): HexString;
