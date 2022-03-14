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
    DeferredOffenceOf: '(Vec<OffenceDetails>, Vec<Perbill>, SessionIndex)',
    Kind: '[u8; 16]',
    OffenceDetails: {
      offender: 'Offender',
      reporters: 'Vec<Reporter>'
    },
    Offender: 'IdentificationTuple',
    OpaqueTimeSlot: 'Bytes',
    ReportIdOf: 'Hash',
    Reporter: 'AccountId'
  }
};
exports.default = _default;