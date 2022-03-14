// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { TextDecoder } from '@polkadot/x-textdecoder';
const decoder = new TextDecoder('utf-8');
/**
 * @name u8aToString
 * @summary Creates a utf-8 string from a Uint8Array object.
 * @description
 * `UInt8Array` input values return the actual decoded utf-8 string. `null` or `undefined` values returns an empty string.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aToString } from '@polkadot/util';
 *
 * u8aToString(new Uint8Array([0x68, 0x65, 0x6c, 0x6c, 0x6f])); // hello
 * ```
 */

export function u8aToString(value) {
  return !(value !== null && value !== void 0 && value.length) ? '' : decoder.decode(value);
}