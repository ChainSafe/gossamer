"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isKeyringPair = isKeyringPair;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
function isKeyringPair(account) {
  return (0, _util.isFunction)(account.sign);
}