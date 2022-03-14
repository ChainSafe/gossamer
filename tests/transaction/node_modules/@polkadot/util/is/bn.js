// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BN } from "../bn/bn.js";
/**
 * @name isBn
 * @summary Tests for a `BN` object instance.
 * @description
 * Checks to see if the input object is an instance of `BN` (bn.js).
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { isBn } from '@polkadot/util';
 *
 * console.log('isBn', isBn(new BN(1))); // => true
 * ```
 */

export function isBn(value) {
  return BN.isBN(value);
}