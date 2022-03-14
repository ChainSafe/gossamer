"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ed25519PairFromSecret = ed25519PairFromSecret;

var _tweetnacl = _interopRequireDefault(require("tweetnacl"));

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name ed25519PairFromSecret
 * @summary Creates a new public/secret keypair from a secret.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied secret.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromSecret } from '@polkadot/util-crypto';
 *
 * ed25519PairFromSecret(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
function ed25519PairFromSecret(secret) {
  return _tweetnacl.default.sign.keyPair.fromSecretKey(secret);
}