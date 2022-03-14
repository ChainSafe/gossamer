"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.findCall = findCall;
exports.findError = findError;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
function findCall(registry, callIndex) {
  return registry.findMetaCall((0, _util.u8aToU8a)(callIndex));
}

function findError(registry, errorIndex) {
  return registry.findMetaError((0, _util.u8aToU8a)(errorIndex));
}