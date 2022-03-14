// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { assert } from "../assert.js";
import { BN } from "./bn.js";
export function find(items, cmp) {
  assert(items.length >= 1, 'Must provide one or more BN arguments');
  let result = items[0];

  for (let i = 1; i < items.length; i++) {
    result = cmp(result, items[i]);
  }

  return result;
}
/**
 * @name bnMax
 * @summary Finds and returns the highest value in an array of BNs.
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnMax } from '@polkadot/util';
 *
 * bnMax([new BN(1), new BN(3), new BN(2)]).toString(); // => '3'
 * ```
 */

export function bnMax(...items) {
  return find(items, BN.max);
}
/**
 * @name bnMin
 * @summary Finds and returns the smallest value in an array of BNs.
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnMin } from '@polkadot/util';
 *
 * bnMin([new BN(1), new BN(3), new BN(2)]).toString(); // => '1'
 * ```
 */

export function bnMin(...items) {
  return find(items, BN.min);
}