"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.BigInt = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _xGlobal = require("@polkadot/x-global");

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-bigint authors & contributors
// SPDX-License-Identifier: Apache-2.0
const BigInt = typeof _xGlobal.xglobal.BigInt === 'function' && typeof _xGlobal.xglobal.BigInt.asIntN === 'function' ? _xGlobal.xglobal.BigInt : () => Number.NaN;
exports.BigInt = BigInt;