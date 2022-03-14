// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isString } from '@polkadot/util';
export function typeToConstructor(registry, type) {
  return isString(type) ? registry.createClassUnsafe(type) : type;
}