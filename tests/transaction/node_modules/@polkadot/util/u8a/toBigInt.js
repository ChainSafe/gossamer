// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
import { _1n } from "../bi/consts.js";
import { objectSpread } from "../object/spread.js";
const U8_MAX = BigInt(256);
const U16_MAX = BigInt(256 * 256);

function xor(input) {
  const result = new Uint8Array(input.length);
  const dvI = new DataView(input.buffer, input.byteOffset);
  const dvO = new DataView(result.buffer);
  const mod = input.length % 2;
  const length = input.length - mod;

  for (let i = 0; i < length; i += 2) {
    dvO.setUint16(i, dvI.getUint16(i) ^ 0xffff);
  }

  if (mod) {
    dvO.setUint8(length, dvI.getUint8(length) ^ 0xff);
  }

  return result;
}

function toBigInt(input) {
  const dvI = new DataView(input.buffer, input.byteOffset);
  const mod = input.length % 2;
  const length = input.length - mod;
  let result = BigInt(0);

  for (let i = 0; i < length; i += 2) {
    result = result * U16_MAX + BigInt(dvI.getUint16(i));
  }

  if (mod) {
    result = result * U8_MAX + BigInt(dvI.getUint8(length));
  }

  return result;
}
/**
 * @name u8aToBigInt
 * @summary Creates a BigInt from a Uint8Array object.
 */


export function u8aToBigInt(value, options = {}) {
  if (!value || !value.length) {
    return BigInt(0);
  }

  const {
    isLe,
    isNegative
  } = objectSpread({
    isLe: true,
    isNegative: false
  }, options);
  const u8a = isLe ? value.reverse() : value;
  return isNegative ? toBigInt(xor(u8a)) * -_1n - _1n : toBigInt(u8a);
}