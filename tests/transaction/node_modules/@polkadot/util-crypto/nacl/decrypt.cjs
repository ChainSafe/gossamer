"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.naclDecrypt = naclDecrypt;

var _tweetnacl = _interopRequireDefault(require("tweetnacl"));

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name naclDecrypt
 * @summary Decrypts a message using the supplied secretKey and nonce
 * @description
 * Returns an decrypted message, using the `secret` and `nonce`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclDecrypt } from '@polkadot/util-crypto';
 *
 * naclDecrypt([...], [...], [...]); // => [...]
 * ```
 */
function naclDecrypt(encrypted, nonce, secret) {
  return _tweetnacl.default.secretbox.open(encrypted, nonce, secret) || null;
}