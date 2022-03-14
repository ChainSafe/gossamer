"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.naclBoxPairFromSecret = naclBoxPairFromSecret;

var _tweetnacl = _interopRequireDefault(require("tweetnacl"));

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name naclBoxPairFromSecret
 * @summary Creates a new public/secret box keypair from a secret.
 * @description
 * Returns a object containing a box `publicKey` & `secretKey` generated from the supplied secret.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclBoxPairFromSecret } from '@polkadot/util-crypto';
 *
 * naclBoxPairFromSecret(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
function naclBoxPairFromSecret(secret) {
  return _tweetnacl.default.box.keyPair.fromSecretKey(secret.slice(0, 32));
}