"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TextDecoder = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _util = _interopRequireDefault(require("util"));

var _xGlobal = require("@polkadot/x-global");

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-textencoder authors & contributors
// SPDX-License-Identifier: Apache-2.0
const TextDecoder = (0, _xGlobal.extractGlobal)('TextDecoder', _util.default.TextDecoder);
exports.TextDecoder = TextDecoder;