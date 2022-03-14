// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isOnObject } from "./helpers.js";
const checker = isOnObject('toBigInt', 'toBn', 'toNumber', 'unwrap');
/**
 * @name isCompact
 * @summary Tests for SCALE-Compact-like object instance.
 */

export function isCompact(value) {
  return checker(value);
}