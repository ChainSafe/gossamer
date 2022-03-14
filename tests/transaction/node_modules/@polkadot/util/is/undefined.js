// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isUndefined
 * @summary Tests for a `undefined` values.
 * @description
 * Checks to see if the input value is `undefined`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isUndefined } from '@polkadot/util';
 *
 * console.log('isUndefined', isUndefined(void(0))); // => true
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function isUndefined(value) {
  return typeof value === 'undefined';
}