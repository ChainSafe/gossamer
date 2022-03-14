import type { AnyString } from '../types';
/**
 * @name isString
 * @summary Tests for a string.
 * @description
 * Checks to see if the input value is a JavaScript string.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isString } from '@polkadot/util';
 *
 * console.log('isString', isString('test')); // => true
 * ```
 */
export declare function isString(value: unknown): value is AnyString;
