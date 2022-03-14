"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isFunction = isFunction;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
// eslint-disable-next-line @typescript-eslint/ban-types

/**
 * @name isFunction
 * @summary Tests for a `function`.
 * @description
 * Checks to see if the input value is a JavaScript function.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isFunction } from '@polkadot/util';
 *
 * isFunction(() => false); // => true
 * ```
 */
function isFunction(value) {
  return typeof value === 'function';
}