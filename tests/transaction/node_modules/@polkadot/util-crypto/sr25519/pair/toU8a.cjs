"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.sr25519KeypairToU8a = sr25519KeypairToU8a;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
function sr25519KeypairToU8a(_ref) {
  let {
    publicKey,
    secretKey
  } = _ref;
  return (0, _util.u8aConcat)(secretKey, publicKey).slice();
}