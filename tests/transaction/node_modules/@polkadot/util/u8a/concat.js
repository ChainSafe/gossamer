// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aToU8a } from "./toU8a.js";
/**
 * @name u8aConcat
 * @summary Creates a concatenated Uint8Array from the inputs.
 * @description
 * Concatenates the input arrays into a single `UInt8Array`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { { u8aConcat } from '@polkadot/util';
 *
 * u8aConcat(
 *   new Uint8Array([1, 2, 3]),
 *   new Uint8Array([4, 5, 6])
 * ); // [1, 2, 3, 4, 5, 6]
 * ```
 */

export function u8aConcat(...list) {
  let length = 0;
  let offset = 0;
  const u8as = new Array(list.length);

  for (let i = 0; i < list.length; i++) {
    u8as[i] = u8aToU8a(list[i]);
    length += u8as[i].length;
  }

  const result = new Uint8Array(length);

  for (let i = 0; i < u8as.length; i++) {
    result.set(u8as[i], offset);
    offset += u8as[i].length;
  }

  return result;
}