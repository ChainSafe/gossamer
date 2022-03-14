"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.exposeGlobal = exposeGlobal;
exports.extractGlobal = extractGlobal;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});
exports.xglobal = void 0;

var _packageInfo = require("./packageInfo.cjs");

// Copyright 2017-2022 @polkadot/x-global authors & contributors
// SPDX-License-Identifier: Apache-2.0
function evaluateThis(fn) {
  return fn('return this');
}

const xglobal = typeof globalThis !== 'undefined' ? globalThis : typeof global !== 'undefined' ? global : typeof self !== 'undefined' ? self : typeof window !== 'undefined' ? window : evaluateThis(Function);
exports.xglobal = xglobal;

function extractGlobal(name, fallback) {
  return typeof xglobal[name] === 'undefined' ? fallback : xglobal[name];
}

function exposeGlobal(name, fallback) {
  if (typeof xglobal[name] === 'undefined') {
    xglobal[name] = fallback;
  }
}