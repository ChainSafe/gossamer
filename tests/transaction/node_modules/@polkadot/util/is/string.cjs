"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isString = isString;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isString
 * @summary Tests for a string.
 * @description
 * Checks to see if the input value is a JavaScript string.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isString } from '@polkadot/util';
 *
 * console.log('isString', isString('test')); // => true
 * ```
 */
function isString(value) {
  return typeof value === 'string' || value instanceof String;
}