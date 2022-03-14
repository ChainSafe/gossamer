"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.WebSocket = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _websocket = _interopRequireDefault(require("websocket"));

var _xGlobal = require("@polkadot/x-global");

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-ws authors & contributors
// SPDX-License-Identifier: Apache-2.0
const WebSocket = (0, _xGlobal.extractGlobal)('WebSocket', _websocket.default.w3cwebsocket);
exports.WebSocket = WebSocket;