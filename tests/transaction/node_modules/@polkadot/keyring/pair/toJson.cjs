"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.pairToJson = pairToJson;

var _util = require("@polkadot/util");

var _utilCrypto = require("@polkadot/util-crypto");

// Copyright 2017-2022 @polkadot/keyring authors & contributors
// SPDX-License-Identifier: Apache-2.0
function pairToJson(type, _ref, encoded, isEncrypted) {
  let {
    address,
    meta
  } = _ref;
  return (0, _util.objectSpread)((0, _utilCrypto.jsonEncryptFormat)(encoded, ['pkcs8', type], isEncrypted), {
    address,
    meta
  });
}