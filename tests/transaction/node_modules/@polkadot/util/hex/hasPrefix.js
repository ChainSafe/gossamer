// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isHex } from "../is/hex.js";
/**
 * @name hexHasPrefix
 * @summary Tests for the existence of a `0x` prefix.
 * @description
 * Checks for a valid hex input value and if the start matched `0x`
 * @example
 * <BR>
 *
 * ```javascript
 * import { hexHasPrefix } from '@polkadot/util';
 *
 * console.log('has prefix', hexHasPrefix('0x1234')); // => true
 * ```
 */

export function hexHasPrefix(value) {
  return !!value && isHex(value, -1);
}