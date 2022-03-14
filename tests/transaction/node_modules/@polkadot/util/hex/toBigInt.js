// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
import { objectSpread } from "../object/spread.js";
import { u8aToBigInt } from "../u8a/toBigInt.js";
import { hexToU8a } from "./toU8a.js";
/**
 * @name hexToBigInt
 * @summary Creates a BigInt instance object from a hex string.
 */

export function hexToBigInt(value, options = {}) {
  return !value || value === '0x' ? BigInt(0) : u8aToBigInt(hexToU8a(value), objectSpread({
    isLe: false,
    isNegative: false
  }, options));
}