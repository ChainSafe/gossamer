"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.createClass = createClass;

var _typesCreate = require("@polkadot/types-create");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
function createClass(registry, type) {
  return (0, _typesCreate.createClassUnsafe)(registry, type);
}