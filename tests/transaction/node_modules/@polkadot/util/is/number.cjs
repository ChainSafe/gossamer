"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isNumber = isNumber;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isNumber
 * @summary Tests for a JavaScript number.
 * @description
 * Checks to see if the input value is a valid number.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isNumber } from '@polkadot/util';
 *
 * console.log('isNumber', isNumber(1234)); // => true
 * ```
 */
function isNumber(value) {
  return typeof value === 'number';
}