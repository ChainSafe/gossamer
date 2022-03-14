// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { stringify } from "../stringify.js";

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
export function isJsonObject(value) {
  const str = typeof value !== 'string' ? stringify(value) : value;

  try {
    const obj = JSON.parse(str);
    return typeof obj === 'object' && obj !== null;
  } catch (e) {
    return false;
  }
}