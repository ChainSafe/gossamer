// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aConcat } from "../u8a/index.js";
import { compactToU8a } from "./toU8a.js";
/**
 * @name compactAddLength
 * @description Adds a length prefix to the input value
 * @example
 * <BR>
 *
 * ```javascript
 * import { compactAddLength } from '@polkadot/util';
 *
 * console.log(compactAddLength(new Uint8Array([0xde, 0xad, 0xbe, 0xef]))); // Uint8Array([4 << 2, 0xde, 0xad, 0xbe, 0xef])
 * ```
 */

export function compactAddLength(input) {
  return u8aConcat(compactToU8a(input.length), input);
}