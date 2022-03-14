// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
import { objectSpread } from "../object/spread.js";
import { _0n, _1n } from "./consts.js";
import { nToBigInt } from "./toBigInt.js";
const DIV = BigInt(256);
const NEG_MASK = BigInt(0xff);

function createEmpty({
  bitLength = 0
}) {
  return bitLength === -1 ? new Uint8Array() : new Uint8Array(Math.ceil(bitLength / 8));
}

function toU8a(value, {
  isLe,
  isNegative
}) {
  const arr = [];

  if (isNegative) {
    value = (value + _1n) * -_1n;
  }

  while (value !== _0n) {
    const mod = value % DIV;
    const val = Number(isNegative ? mod ^ NEG_MASK : mod);

    if (isLe) {
      arr.push(val);
    } else {
      arr.unshift(val);
    }

    value = (value - mod) / DIV;
  }

  return Uint8Array.from(arr);
}
/**
 * @name nToU8a
 * @summary Creates a Uint8Array object from a bigint.
 */


export function nToU8a(value, options) {
  const opts = objectSpread({
    bitLength: -1,
    isLe: true,
    isNegative: false
  }, options);
  const valueBi = nToBigInt(value);

  if (valueBi === _0n) {
    return createEmpty(opts);
  }

  const u8a = toU8a(valueBi, opts);

  if (opts.bitLength === -1) {
    return u8a;
  }

  const byteLength = Math.ceil((opts.bitLength || 0) / 8);
  const output = new Uint8Array(byteLength);

  if (opts.isNegative) {
    output.fill(0xff);
  }

  output.set(u8a, opts.isLe ? 0 : byteLength - u8a.length);
  return output;
}