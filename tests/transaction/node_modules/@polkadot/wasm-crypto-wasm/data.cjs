"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.wasmBytes = void 0;

var _base = require("./base64.cjs");

var _bytes = require("./bytes.cjs");

var _fflate = require("./fflate.cjs");

// Copyright 2019-2021 @polkadot/wasm-crypto-wasm authors & contributors
// SPDX-License-Identifier: Apache-2.0
const wasmBytes = (0, _fflate.unzlibSync)((0, _base.base64Decode)(_bytes.bytes), new Uint8Array(_bytes.sizeUncompressed));
exports.wasmBytes = wasmBytes;