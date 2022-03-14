// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { hexToBn } from "./toBn.js";
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

export function hexToNumber(value) {
  return value ? hexToBn(value).toNumber() : NaN;
}