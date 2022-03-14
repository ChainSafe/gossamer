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
    Multisig: {
      when: 'Timepoint',
      deposit: 'Balance',
      depositor: 'AccountId',
      approvals: 'Vec<AccountId>'
    },
    Timepoint: {
      height: 'BlockNumber',
      index: 'u32'
    }
  }
};
exports.default = _default;