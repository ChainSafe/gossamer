"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.stringShorten = stringShorten;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name stringShorten
 * @summary Returns a string with maximum length
 * @description
 * Checks the string against the `prefixLength`, if longer than double this, shortens it by placing `..` in the middle of it
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringShorten } from '@polkadot/util';
 *
 * stringShorten('1234567890', 2); // => 12..90
 * ```
 */
function stringShorten(value) {
  let prefixLength = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : 6;
  return value.length <= 2 + 2 * prefixLength ? value.toString() : `${value.substr(0, prefixLength)}…${value.slice(-prefixLength)}`;
}