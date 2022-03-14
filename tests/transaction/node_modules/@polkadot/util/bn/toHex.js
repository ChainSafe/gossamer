// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isNumber } from "../is/number.js";
import { objectSpread } from "../object/spread.js";
import { u8aToHex } from "../u8a/index.js";
import { bnToU8a } from "./toU8a.js";
const ZERO_STR = '0x00';
const DEFAULT_OPTS = {
  bitLength: -1,
  isLe: false,
  isNegative: false
};
/**
 * @name bnToHex
 * @summary Creates a hex value from a BN.js bignumber object.
 * @description
 * `null` inputs returns a `0x` result, BN values return the actual value as a `0x` prefixed hex value. Anything that is not a BN object throws an error. With `bitLength` set, it fixes the number to the specified length.
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnToHex } from '@polkadot/util';
 *
 * bnToHex(new BN(0x123456)); // => '0x123456'
 * ```
 */

function bnToHex(value, arg1 = DEFAULT_OPTS, arg2) {
  return !value ? ZERO_STR : u8aToHex(bnToU8a(value, objectSpread( // We spread here, the default for hex values is BE (JSONRPC via substrate)
  {
    isLe: false,
    isNegative: false
  }, isNumber(arg1) ? {
    bitLength: arg1,
    isLe: arg2
  } : arg1)));
}

export { bnToHex };