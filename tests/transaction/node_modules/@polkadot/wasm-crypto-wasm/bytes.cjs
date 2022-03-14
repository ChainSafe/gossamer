"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.sizeUncompressed = exports.sizeCompressed = exports.bytes = void 0;

var _bytes2 = require("./cjs/bytes");

// Copyright 2019-2021 @polkadot/wasm-crypto-wasm authors & contributors
// SPDX-License-Identifier: Apache-2.0
const bytes = _bytes2.bytes;
exports.bytes = bytes;
const sizeCompressed = _bytes2.sizeCompressed;
exports.sizeCompressed = sizeCompressed;
const sizeUncompressed = _bytes2.sizeUncompressed;
exports.sizeUncompressed = sizeUncompressed;