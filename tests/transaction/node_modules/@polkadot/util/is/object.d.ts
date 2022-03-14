interface ObjectIndexed {
    [index: string]: any;
}
/**
 * @name isObject
 * @summary Tests for an `object`.
 * @description
 * Checks to see if the input value is a JavaScript object.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isObject } from '@polkadot/util';
 *
 * isObject({}); // => true
 * isObject('something'); // => false
 * ```
 */
export declare function isObject<T extends ObjectIndexed = ObjectIndexed>(value?: unknown): value is T;
export {};
