/**
 * @name isUndefined
 * @summary Tests for a `undefined` values.
 * @description
 * Checks to see if the input value is `undefined`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isUndefined } from '@polkadot/util';
 *
 * console.log('isUndefined', isUndefined(void(0))); // => true
 * ```
 */
export declare function isUndefined(value?: unknown): value is undefined;
