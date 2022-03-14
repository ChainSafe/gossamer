// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/** @internal */
export function flattenUniq(list, result = []) {
  for (let i = 0; i < list.length; i++) {
    const entry = list[i];

    if (Array.isArray(entry)) {
      flattenUniq(entry, result);
    } else {
      result.push(entry);
    }
  }

  return [...new Set(result)];
}