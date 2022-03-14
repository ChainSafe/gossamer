"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isError = isError;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isError
 * @summary Tests for a `Error` object instance.
 * @description
 * Checks to see if the input object is an instance of `Error`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isError } from '@polkadot/util';
 *
 * console.log('isError', isError(new Error('message'))); // => true
 * ```
 */
function isError(value) {
  return value instanceof Error;
}