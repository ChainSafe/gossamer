"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isObject = isObject;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isObject
 * @summary Tests for an `object`.
 * @description
 * Checks to see if the input value is a JavaScript object.
 * @example
 * <BR>
 *
 * ```javascript
 * import { isObject } from '@polkadot/util';
 *
 * isObject({}); // => true
 * isObject('something'); // => false
 * ```
 */
function isObject(value) {
  return !!value && typeof value === 'object';
}