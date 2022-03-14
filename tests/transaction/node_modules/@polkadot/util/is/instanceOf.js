// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isInstanceOf
 * @summary Tests for a instance of a class.
 * @description
 * Checks to see if the input value is an instance of the test class.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isInstanceOf } from '@polkadot/util';
 *
 * console.log('isInstanceOf', isInstanceOf(new Array(0), Array)); // => true
 * ```
 */
// eslint-disable-next-line @typescript-eslint/ban-types
export function isInstanceOf(value, clazz) {
  return value instanceof clazz;
}