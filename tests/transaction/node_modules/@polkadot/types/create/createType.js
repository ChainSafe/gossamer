// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { createTypeUnsafe } from '@polkadot/types-create';
/**
 * Create an instance of a `type` with a given `params`.
 * @param type - A recognizable string representing the type to create an
 * instance from
 * @param params - The value to instantiate the type with
 */

export function createType(registry, type, ...params) {
  return createTypeUnsafe(registry, type, params);
}