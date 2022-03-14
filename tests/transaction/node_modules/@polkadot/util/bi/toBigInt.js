// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
import { hexToBigInt } from "../hex/toBigInt.js";
import { isBn } from "../is/bn.js";
import { isHex } from "../is/hex.js";
import { isToBigInt } from "../is/toBigInt.js";
import { isToBn } from "../is/toBn.js";
/**
 * @name nToBigInt
 * @summary Creates a bigInt value from a BN, bigint, string (base 10 or hex) or number input.
 */

export function nToBigInt(value) {
  return typeof value === 'bigint' ? value : !value ? BigInt(0) : isHex(value) ? hexToBigInt(value.toString()) : isBn(value) ? BigInt(value.toString()) : isToBigInt(value) ? value.toBigInt() : isToBn(value) ? BigInt(value.toBn().toString()) : BigInt(value);
}