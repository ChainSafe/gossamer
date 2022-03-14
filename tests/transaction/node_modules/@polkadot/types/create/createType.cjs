"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.createType = createType;

var _typesCreate = require("@polkadot/types-create");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * Create an instance of a `type` with a given `params`.
 * @param type - A recognizable string representing the type to create an
 * instance from
 * @param params - The value to instantiate the type with
 */
function createType(registry, type) {
  for (var _len = arguments.length, params = new Array(_len > 2 ? _len - 2 : 0), _key = 2; _key < _len; _key++) {
    params[_key - 2] = arguments[_key];
  }

  return (0, _typesCreate.createTypeUnsafe)(registry, type, params);
}