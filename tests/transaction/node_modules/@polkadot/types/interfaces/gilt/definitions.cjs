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
    ActiveGilt: {
      proportion: 'Perquintill',
      amount: 'Balance',
      who: 'AccountId',
      expiry: 'BlockNumber'
    },
    ActiveGiltsTotal: {
      frozen: 'Balance',
      proportion: 'Perquintill',
      index: 'ActiveIndex',
      target: 'Perquintill'
    },
    ActiveIndex: 'u32',
    GiltBid: {
      amount: 'Balance',
      who: 'AccountId'
    }
  }
};
exports.default = _default;