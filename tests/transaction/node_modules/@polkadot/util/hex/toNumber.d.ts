/**
 * @name hexToNumber
 * @summary Creates a Number value from a Buffer object.
 * @description
 * `null` inputs returns an NaN result, `hex` values return the actual value as a `Number`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { hexToNumber } from '@polkadot/util';
 *
 * hexToNumber('0x1234'); // => 0x1234
 * ```
 */
export declare function hexToNumber(value?: string | null): number;
