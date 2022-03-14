// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { HEX_TO_U8, HEX_TO_U16 } from "./alphabet.js";
import { hexStripPrefix } from "./stripPrefix.js";
/**
 * @name hexToU8a
 * @summary Creates a Uint8Array object from a hex string.
 * @description
 * `null` inputs returns an empty `Uint8Array` result. Hex input values return the actual bytes value converted to a Uint8Array. Anything that is not a hex string (including the `0x` prefix) throws an error.
 * @example
 * <BR>
 *
 * ```javascript
 * import { hexToU8a } from '@polkadot/util';
 *
 * hexToU8a('0x80001f'); // Uint8Array([0x80, 0x00, 0x1f])
 * hexToU8a('0x80001f', 32); // Uint8Array([0x00, 0x80, 0x00, 0x1f])
 * ```
 */

export function hexToU8a(_value, bitLength = -1) {
  if (!_value) {
    return new Uint8Array();
  }

  const value = hexStripPrefix(_value).toLowerCase();
  const valLength = value.length / 2;
  const endLength = Math.ceil(bitLength === -1 ? valLength : bitLength / 8);
  const result = new Uint8Array(endLength);
  const offset = endLength > valLength ? endLength - valLength : 0;
  const dv = new DataView(result.buffer, offset);
  const mod = (endLength - offset) % 2;
  const length = endLength - offset - mod;

  for (let i = 0; i < length; i += 2) {
    dv.setUint16(i, HEX_TO_U16[value.substr(i * 2, 4)]);
  }

  if (mod) {
    dv.setUint8(length, HEX_TO_U8[value.substr(value.length - 2, 2)]);
  }

  return result;
}