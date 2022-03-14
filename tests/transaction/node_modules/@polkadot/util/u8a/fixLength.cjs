"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.u8aFixLength = u8aFixLength;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name u8aFixLength
 * @summary Shifts a Uint8Array to a specific bitLength
 * @description
 * Returns a uint8Array with the specified number of bits contained in the return value. (If bitLength is -1, length checking is not done). Values with more bits are trimmed to the specified length.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aFixLength } from '@polkadot/util';
 *
 * u8aFixLength('0x12') // => 0x12
 * u8aFixLength('0x12', 16) // => 0x0012
 * u8aFixLength('0x1234', 8) // => 0x12
 * ```
 */
function u8aFixLength(value) {
  let bitLength = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : -1;
  let atStart = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : false;
  const byteLength = Math.ceil(bitLength / 8);

  if (bitLength === -1 || value.length === byteLength) {
    return value;
  } else if (value.length > byteLength) {
    return value.subarray(0, byteLength);
  }

  const result = new Uint8Array(byteLength);
  result.set(value, atStart ? 0 : byteLength - value.length);
  return result;
}