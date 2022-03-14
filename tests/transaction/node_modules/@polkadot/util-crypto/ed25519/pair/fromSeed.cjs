"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ed25519PairFromSeed = ed25519PairFromSeed;

var _tweetnacl = _interopRequireDefault(require("tweetnacl"));

var _wasmCrypto = require("@polkadot/wasm-crypto");

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name ed25519PairFromSeed
 * @summary Creates a new public/secret keypair from a seed.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied seed.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromSeed } from '@polkadot/util-crypto';
 *
 * ed25519PairFromSeed(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
function ed25519PairFromSeed(seed, onlyJs) {
  if (!onlyJs && (0, _wasmCrypto.isReady)()) {
    const full = (0, _wasmCrypto.ed25519KeypairFromSeed)(seed);
    return {
      publicKey: full.slice(32),
      secretKey: full.slice(0, 64)
    };
  }

  return _tweetnacl.default.sign.keyPair.fromSeed(seed);
}