import type { HexString } from '../types';
export declare const REGEX_HEX_PREFIXED: RegExp;
export declare const REGEX_HEX_NOPREFIX: RegExp;
/**
 * @name isHex
 * @summary Tests for a hex string.
 * @description
 * Checks to see if the input value is a `0x` prefixed hex string. Optionally (`bitLength` !== -1) checks to see if the bitLength is correct.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isHex } from '@polkadot/util';
 *
 * isHex('0x1234'); // => true
 * isHex('0x1234', 8); // => false
 * ```
 */
export declare function isHex(value: unknown, bitLength?: number, ignoreLength?: boolean): value is HexString;
