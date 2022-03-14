// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
// This is supposed to be a faster concat...
// https://dev.to/uilicious/javascript-array-push-is-945x-faster-than-array-concat-1oki

/**
 * @name arrayFlatten
 * @summary Merge T[][] into T[]
 * @description
 * Returns a new array with all arrays merged into one
 * @example
 * <BR>
 *
 * ```javascript
 * import { arrayFlatten } from '@polkadot/util';
 *
 * arrayFlatten([[1, 2], [3, 4], [5]]); // [1, 2, 3, 4, 5]
 * ```
 */
export function arrayFlatten(arrays) {
  // pre-allocate based on the combined size
  let size = 0;

  for (let i = 0; i < arrays.length; i++) {
    size += arrays[i].length;
  }

  const output = new Array(size);
  let i = -1;

  for (let j = 0; j < arrays.length; j++) {
    const a = arrays[j]; // instead of pushing, we just set the entries

    for (let e = 0; e < a.length; e++) {
      output[++i] = a[e];
    }
  }

  return output;
}