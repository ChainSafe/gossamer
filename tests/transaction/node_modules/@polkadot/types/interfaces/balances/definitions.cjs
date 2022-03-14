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
    AccountData: {
      free: 'Balance',
      reserved: 'Balance',
      miscFrozen: 'Balance',
      feeFrozen: 'Balance'
    },
    BalanceLockTo212: {
      id: 'LockIdentifier',
      amount: 'Balance',
      until: 'BlockNumber',
      reasons: 'WithdrawReasons'
    },
    BalanceLock: {
      id: 'LockIdentifier',
      amount: 'Balance',
      reasons: 'Reasons'
    },
    BalanceStatus: {
      _enum: ['Free', 'Reserved']
    },
    Reasons: {
      _enum: ['Fee', 'Misc', 'All']
    },
    ReserveData: {
      id: 'ReserveIdentifier',
      amount: 'Balance'
    },
    ReserveIdentifier: '[u8; 8]',
    VestingSchedule: {
      offset: 'Balance',
      perBlock: 'Balance',
      startingBlock: 'BlockNumber'
    },
    WithdrawReasons: {
      _set: {
        TransactionPayment: 0b00000001,
        Transfer: 0b00000010,
        Reserve: 0b00000100,
        Fee: 0b00001000,
        Tip: 0b00010000
      }
    }
  }
};
exports.default = _default;