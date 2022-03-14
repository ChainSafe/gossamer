interface ObjectIndexed {
    [index: string]: any;
}
/**
 * @name isJsonObject
 * @summary Tests for a valid JSON `object`.
 * @description
 * Checks to see if the input value is a valid JSON object.
 * It returns false if the input is JSON parsable, but not an Javascript object.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isJsonObject } from '@polkadot/util';
 *
 * isJsonObject({}); // => true
 * isJsonObject({
 *  "Test": "1234",
 *  "NestedTest": {
 *   "Test": "5678"
 *  }
 * }); // => true
 * isJsonObject(1234); // JSON parsable, but not an object =>  false
 * isJsonObject(null); // JSON parsable, but not an object => false
 * isJsonObject('not an object'); // => false
 * ```
 */
export declare function isJsonObject(value: unknown): value is ObjectIndexed;
export {};
