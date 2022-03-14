import type { AnyString, HexString } from '../types';
/**
 * @name stringToHex
 * @summary Creates a hex string from a utf-8 string
 * @description
 * String input values return the actual encoded hex value.
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringToHex } from '@polkadot/util';
 *
 * stringToU8a('hello'); // 0x68656c6c6f
 * ```
 */
export declare function stringToHex(value?: AnyString): HexString;
