// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { createClassUnsafe } from '@polkadot/types-create';
export function createClass(registry, type) {
  return createClassUnsafe(registry, type);
}