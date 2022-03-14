"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.hmacSha512AsU8a = exports.hmacSha256AsU8a = void 0;
exports.hmacShaAsU8a = hmacShaAsU8a;

var _hmac = require("@noble/hashes/hmac");

var _sha = require("@noble/hashes/sha256");

var _sha2 = require("@noble/hashes/sha512");

var _util = require("@polkadot/util");

var _wasmCrypto = require("@polkadot/wasm-crypto");

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const JS_HASH = {
  256: _sha.sha256,
  512: _sha2.sha512
};
const WA_MHAC = {
  256: _wasmCrypto.hmacSha256,
  512: _wasmCrypto.hmacSha512
};

function createSha(bitLength) {
  return (key, data, onlyJs) => hmacShaAsU8a(key, data, bitLength, onlyJs);
}
/**
 * @name hmacShaAsU8a
 * @description creates a Hmac Sha (256/512) Uint8Array from the key & data
 */


function hmacShaAsU8a(key, data) {
  let bitLength = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : 256;
  let onlyJs = arguments.length > 3 ? arguments[3] : undefined;
  const u8aKey = (0, _util.u8aToU8a)(key);
  return !_util.hasBigInt || !onlyJs && (0, _wasmCrypto.isReady)() ? WA_MHAC[bitLength](u8aKey, data) : (0, _hmac.hmac)(JS_HASH[bitLength], u8aKey, data);
}
/**
 * @name hmacSha256AsU8a
 * @description creates a Hmac Sha256 Uint8Array from the key & data
 */


const hmacSha256AsU8a = createSha(256);
/**
 * @name hmacSha512AsU8a
 * @description creates a Hmac Sha512 Uint8Array from the key & data
 */

exports.hmacSha256AsU8a = hmacSha256AsU8a;
const hmacSha512AsU8a = createSha(512);
exports.hmacSha512AsU8a = hmacSha512AsU8a;