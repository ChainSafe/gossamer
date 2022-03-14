// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { U8_TO_HEX, U16_TO_HEX } from "../hex/alphabet.js";
/** @internal */

function hex(value) {
  const mod = value.length % 2;
  const length = value.length - mod;
  const dv = new DataView(value.buffer, value.byteOffset);
  let result = '';

  for (let i = 0; i < length; i += 2) {
    result += U16_TO_HEX[dv.getUint16(i)];
  }

  if (mod) {
    result += U8_TO_HEX[dv.getUint8(length)];
  }

  return result;
}
/**
 * @name u8aToHex
 * @summary Creates a hex string from a Uint8Array object.
 * @description
 * `UInt8Array` input values return the actual hex string. `null` or `undefined` values returns an `0x` string.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aToHex } from '@polkadot/util';
 *
 * u8aToHex(new Uint8Array([0x68, 0x65, 0x6c, 0x6c, 0xf])); // 0x68656c0f
 * ```
 */


export function u8aToHex(value, bitLength = -1, isPrefixed = true) {
  const length = Math.ceil(bitLength / 8);
  return `${isPrefixed ? '0x' : ''}${!value || !value.length ? '' : length > 0 && value.length > length ? `${hex(value.subarray(0, length / 2))}…${hex(value.subarray(value.length - length / 2))}` : hex(value)}`;
}