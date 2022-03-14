"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.stringUpperFirst = exports.stringLowerFirst = void 0;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
function converter(fn) {
  return value => value ? fn(value[0]) + value.slice(1) : '';
}
/**
 * @name stringLowerFirst
 * @summary Lowercase the first letter of a string
 * @description
 * Lowercase the first letter of a string
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringLowerFirst } from '@polkadot/util';
 *
 * stringLowerFirst('ABC'); // => 'aBC'
 * ```
 */


const stringLowerFirst = converter(s => s.toLowerCase());
/**
 * @name stringUpperFirst
 * @summary Uppercase the first letter of a string
 * @description
 * Lowercase the first letter of a string
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringUpperFirst } from '@polkadot/util';
 *
 * stringUpperFirst('abc'); // => 'Abc'
 * ```
 */

exports.stringLowerFirst = stringLowerFirst;
const stringUpperFirst = converter(s => s.toUpperCase());
exports.stringUpperFirst = stringUpperFirst;