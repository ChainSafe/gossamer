"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.typeToConstructor = typeToConstructor;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
function typeToConstructor(registry, type) {
  return (0, _util.isString)(type) ? registry.createClassUnsafe(type) : type;
}