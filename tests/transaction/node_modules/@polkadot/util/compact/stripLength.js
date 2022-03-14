// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { compactFromU8a } from "./fromU8a.js";
/**
 * @name compactStripLength
 * @description Removes the length prefix, returning both the total length (including the value + compact encoding) and the decoded value with the correct length
 * @example
 * <BR>
 *
 * ```javascript
 * import { compactStripLength } from '@polkadot/util';
 *
 * console.log(compactStripLength(new Uint8Array([2 << 2, 0xde, 0xad]))); // [2, Uint8Array[0xde, 0xad]]
 * ```
 */

export function compactStripLength(input) {
  const [offset, length] = compactFromU8a(input);
  const total = offset + length.toNumber();
  return [total, input.subarray(offset, total)];
}