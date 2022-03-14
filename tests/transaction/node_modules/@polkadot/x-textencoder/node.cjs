"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TextEncoder = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _classPrivateFieldLooseBase2 = _interopRequireDefault(require("@babel/runtime/helpers/classPrivateFieldLooseBase"));

var _classPrivateFieldLooseKey2 = _interopRequireDefault(require("@babel/runtime/helpers/classPrivateFieldLooseKey"));

var _util = _interopRequireDefault(require("util"));

var _xGlobal = require("@polkadot/x-global");

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-textencoder authors & contributors
// SPDX-License-Identifier: Apache-2.0
var _encoder = /*#__PURE__*/(0, _classPrivateFieldLooseKey2.default)("encoder");

class Fallback {
  constructor() {
    Object.defineProperty(this, _encoder, {
      writable: true,
      value: void 0
    });
    (0, _classPrivateFieldLooseBase2.default)(this, _encoder)[_encoder] = new _util.default.TextEncoder();
  } // For a Jest 26.0.1 environment, Buffer !== Uint8Array


  encode(value) {
    return Uint8Array.from((0, _classPrivateFieldLooseBase2.default)(this, _encoder)[_encoder].encode(value));
  }

}

const TextEncoder = (0, _xGlobal.extractGlobal)('TextEncoder', Fallback);
exports.TextEncoder = TextEncoder;