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
    AccountStatus: {
      validity: 'AccountValidity',
      freeBalance: 'Balance',
      lockedBalance: 'Balance',
      signature: 'Vec<u8>',
      vat: 'Permill'
    },
    AccountValidity: {
      _enum: ['Invalid', 'Initiated', 'Pending', 'ValidLow', 'ValidHigh', 'Completed']
    }
  }
};
exports.default = _default;