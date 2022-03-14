// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

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
  return `${isPrefixed ? '0x' : ''}${!value || !value.length ? '' : length > 0 && value.length > length ? `${Buffer.from(value.subarray(0, length / 2)).toString('hex')}…${Buffer.from(value.subarray(value.length - length / 2)).toString('hex')}` : Buffer.from(value).toString('hex')}`;
}