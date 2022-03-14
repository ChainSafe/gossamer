"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.UNMASK_VERSION = exports.IMMORTAL_ERA = exports.EMPTY_U8A = exports.DEFAULT_VERSION = exports.BIT_UNSIGNED = exports.BIT_SIGNED = void 0;
// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
const BIT_SIGNED = 0b10000000;
exports.BIT_SIGNED = BIT_SIGNED;
const BIT_UNSIGNED = 0;
exports.BIT_UNSIGNED = BIT_UNSIGNED;
const EMPTY_U8A = new Uint8Array();
exports.EMPTY_U8A = EMPTY_U8A;
const DEFAULT_VERSION = 4;
exports.DEFAULT_VERSION = DEFAULT_VERSION;
const IMMORTAL_ERA = new Uint8Array([0]);
exports.IMMORTAL_ERA = IMMORTAL_ERA;
const UNMASK_VERSION = 0b01111111;
exports.UNMASK_VERSION = UNMASK_VERSION;