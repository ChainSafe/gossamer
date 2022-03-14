"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isIp = isIp;

var _ipRegex = _interopRequireDefault(require("ip-regex"));

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name isIp
 * @summary Tests if the value is a valid IP address
 * @description
 * Checks to see if the value is a valid IP address. Optionally check for either v4/v6
 * @example
 * <BR>
 *
 * ```javascript
 * import { isIp } from '@polkadot/util';
 *
 * isIp('192.168.0.1')); // => true
 * isIp('1:2:3:4:5:6:7:8'); // => true
 * isIp('192.168.0.1', 'v6')); // => false
 * isIp('1:2:3:4:5:6:7:8', 'v4'); // => false
 * ```
 */
function isIp(value, type) {
  if (type === 'v4') {
    return _ipRegex.default.v4({
      exact: true
    }).test(value);
  } else if (type === 'v6') {
    return _ipRegex.default.v6({
      exact: true
    }).test(value);
  }

  return (0, _ipRegex.default)({
    exact: true
  }).test(value);
}