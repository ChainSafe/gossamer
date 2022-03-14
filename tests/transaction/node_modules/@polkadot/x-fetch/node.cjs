"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.fetch = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _nodeFetch = _interopRequireDefault(require("node-fetch"));

var _xGlobal = require("@polkadot/x-global");

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-fetch authors & contributors
// SPDX-License-Identifier: Apache-2.0
const fetch = (0, _xGlobal.extractGlobal)('fetch', _nodeFetch.default);
exports.fetch = fetch;