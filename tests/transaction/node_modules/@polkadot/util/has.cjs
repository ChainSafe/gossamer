"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.hasWasm = exports.hasProcess = exports.hasEsm = exports.hasDirname = exports.hasCjs = exports.hasBuffer = exports.hasBigInt = void 0;

var _xBigint = require("@polkadot/x-bigint");

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
const hasBigInt = typeof _xBigint.BigInt === 'function' && typeof _xBigint.BigInt.asIntN === 'function';
exports.hasBigInt = hasBigInt;
const hasBuffer = typeof Buffer !== 'undefined';
exports.hasBuffer = hasBuffer;
const hasCjs = typeof require === 'function' && typeof module !== 'undefined';
exports.hasCjs = hasCjs;
const hasDirname = typeof __dirname !== 'undefined';
exports.hasDirname = hasDirname;
const hasEsm = !hasCjs;
exports.hasEsm = hasEsm;
const hasProcess = typeof process === 'object';
exports.hasProcess = hasProcess;
const hasWasm = typeof WebAssembly !== 'undefined';
exports.hasWasm = hasWasm;