// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isOnObject } from "./helpers.js";
const checkCodec = isOnObject('toHex', 'toU8a');
const checkRegistry = isOnObject('get');
export function isCodec(value) {
  return checkCodec(value) && checkRegistry(value.registry);
}