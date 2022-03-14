/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { ToBnOptions } from '../types';
/**
 * @name u8aToBn
 * @summary Creates a BN from a Uint8Array object.
 * @description
 * `UInt8Array` input values return the actual BN. `null` or `undefined` values returns an `0x0` value.
 * @param value The value to convert
 * @param options Options to pass while converting
 * @param options.isLe Convert using Little Endian
 * @param options.isNegative Convert using two's complement
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aToBn } from '@polkadot/util';
 *
 * u8aToHex(new Uint8Array([0x68, 0x65, 0x6c, 0x6c, 0xf])); // 0x68656c0f
 * ```
 */
export declare function u8aToBn(value: Uint8Array, options?: ToBnOptions | boolean): BN;
