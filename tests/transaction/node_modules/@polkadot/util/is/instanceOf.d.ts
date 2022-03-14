/**
 * @name isInstanceOf
 * @summary Tests for a instance of a class.
 * @description
 * Checks to see if the input value is an instance of the test class.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isInstanceOf } from '@polkadot/util';
 *
 * console.log('isInstanceOf', isInstanceOf(new Array(0), Array)); // => true
 * ```
 */
export declare function isInstanceOf(value: unknown, clazz: Function): boolean;
