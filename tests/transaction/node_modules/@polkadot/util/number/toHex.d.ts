import type { HexString } from '../types';
/**
 * @name numberToHex
 * @summary Creates a hex value from a number.
 * @description
 * `null`/`undefined`/`NaN` inputs returns an empty `0x` result. `number` input values return the actual bytes value converted to a `hex`. With `bitLength` set, it converts the number to the equivalent size.
 * @example
 * <BR>
 *
 * ```javascript
 * import { numberToHex } from '@polkadot/util';
 *
 * numberToHex(0x1234); // => '0x1234'
 * numberToHex(0x1234, 32); // => 0x00001234
 * ```
 */
export declare function numberToHex(value?: number | null, bitLength?: number): HexString;
