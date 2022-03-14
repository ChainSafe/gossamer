"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.getRandomValues = getRandomValues;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _crypto = _interopRequireDefault(require("crypto"));

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-randomvalues authors & contributors
// SPDX-License-Identifier: Apache-2.0
function getRandomValues(output) {
  const bytes = _crypto.default.randomBytes(output.length);

  for (let i = 0; i < bytes.length; i++) {
    output[i] = bytes[i];
  }

  return output;
}