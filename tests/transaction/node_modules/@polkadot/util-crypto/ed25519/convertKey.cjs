"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.convertPublicKeyToCurve25519 = convertPublicKeyToCurve25519;
exports.convertSecretKeyToCurve25519 = convertSecretKeyToCurve25519;

var _ed2curve = _interopRequireDefault(require("ed2curve"));

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
function convertSecretKeyToCurve25519(secretKey) {
  return _ed2curve.default.convertSecretKey(secretKey);
}

function convertPublicKeyToCurve25519(publicKey) {
  return (0, _util.assertReturn)(_ed2curve.default.convertPublicKey(publicKey), 'Unable to convert publicKey to ed25519');
}