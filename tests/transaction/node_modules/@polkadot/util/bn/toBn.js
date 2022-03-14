// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { hexToBn } from "../hex/toBn.js";
import { isBigInt } from "../is/bigInt.js";
import { isHex } from "../is/hex.js";
import { isToBigInt } from "../is/toBigInt.js";
import { isToBn } from "../is/toBn.js";
import { BN } from "./bn.js";
/**
 * @name bnToBn
 * @summary Creates a BN value from a BN, bigint, string (base 10 or hex) or number input.
 * @description
 * `null` inputs returns a `0x0` result, BN values returns the value, numbers returns a BN representation.
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnToBn } from '@polkadot/util';
 *
 * bnToBn(0x1234); // => BN(0x1234)
 * bnToBn(new BN(0x1234)); // => BN(0x1234)
 * ```
 */

export function bnToBn(value) {
  return BN.isBN(value) ? value : !value ? new BN(0) : isHex(value) ? hexToBn(value.toString()) : isBigInt(value) ? new BN(value.toString()) : isToBn(value) ? value.toBn() : isToBigInt(value) ? new BN(value.toBigInt().toString()) : new BN(value);
}