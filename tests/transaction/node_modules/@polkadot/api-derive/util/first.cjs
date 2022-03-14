"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.firstMemo = firstMemo;
exports.firstObservable = firstObservable;

var _rxjs = require("rxjs");

var _rpcCore = require("@polkadot/rpc-core");

// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
function firstObservable(obs) {
  return obs.pipe((0, _rxjs.map)(_ref => {
    let [a] = _ref;
    return a;
  }));
}

function firstMemo(fn) {
  return (instanceId, api) => (0, _rpcCore.memo)(instanceId, function () {
    for (var _len = arguments.length, args = new Array(_len), _key = 0; _key < _len; _key++) {
      args[_key] = arguments[_key];
    }

    return firstObservable(fn(api, ...args));
  });
}