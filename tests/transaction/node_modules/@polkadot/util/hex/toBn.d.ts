/// <reference types="bn.js" />
import type { ToBnOptions } from '../types';
import { BN } from '../bn/bn';
/**
 * @name hexToBn
 * @summary Creates a BN.js object from a hex string.
 * @description
 * `null` inputs returns a `BN(0)` result. Hex input values return the actual value converted to a BN. Anything that is not a hex string (including the `0x` prefix) throws an error.
 * @param _value The value to convert
 * @param _options Options to pass while converting
 * @param _options.isLe Convert using Little Endian
 * @param _options.isNegative Convert using two's complement
 * @example
 * <BR>
 *
 * ```javascript
 * import { hexToBn } from '@polkadot/util';
 *
 * hexToBn('0x123480001f'); // => BN(0x123480001f)
 * ```
 */
export declare function hexToBn(value?: string | null, options?: ToBnOptions | boolean): BN;
