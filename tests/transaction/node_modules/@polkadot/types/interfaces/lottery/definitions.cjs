"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
// order important in structs... :)

/* eslint-disable sort-keys */
var _default = {
  rpc: {},
  types: {
    CallIndex: '(u8, u8)',
    LotteryConfig: {
      price: 'Balance',
      start: 'BlockNumber',
      length: 'BlockNumber',
      delay: 'BlockNumber',
      repeat: 'bool'
    }
  }
};
exports.default = _default;