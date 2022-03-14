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
    IdentityFields: {
      _set: {
        _bitLength: 64,
        // Mapped here to 32 bits, in Rust these are 64-bit values
        Display: 0b00000000000000000000000000000001,
        Legal: 0b00000000000000000000000000000010,
        Web: 0b00000000000000000000000000000100,
        Riot: 0b00000000000000000000000000001000,
        Email: 0b00000000000000000000000000010000,
        PgpFingerprint: 0b00000000000000000000000000100000,
        Image: 0b00000000000000000000000001000000,
        Twitter: 0b00000000000000000000000010000000
      }
    },
    IdentityInfoAdditional: '(Data, Data)',
    IdentityInfoTo198: {
      additional: 'Vec<IdentityInfoAdditional>',
      display: 'Data',
      legal: 'Data',
      web: 'Data',
      riot: 'Data',
      email: 'Data',
      pgpFingerprint: 'Option<H160>',
      image: 'Data'
    },
    IdentityInfo: {
      _fallback: 'IdentityInfoTo198',
      additional: 'Vec<IdentityInfoAdditional>',
      display: 'Data',
      legal: 'Data',
      web: 'Data',
      riot: 'Data',
      email: 'Data',
      pgpFingerprint: 'Option<H160>',
      image: 'Data',
      twitter: 'Data'
    },
    IdentityJudgement: {
      _enum: {
        Unknown: 'Null',
        FeePaid: 'Balance',
        Reasonable: 'Null',
        KnownGood: 'Null',
        OutOfDate: 'Null',
        LowQuality: 'Null',
        Erroneous: 'Null'
      }
    },
    RegistrationJudgement: '(RegistrarIndex, IdentityJudgement)',
    RegistrationTo198: {
      judgements: 'Vec<RegistrationJudgement>',
      deposit: 'Balance',
      info: 'IdentityInfoTo198'
    },
    Registration: {
      _fallback: 'RegistrationTo198',
      judgements: 'Vec<RegistrationJudgement>',
      deposit: 'Balance',
      info: 'IdentityInfo'
    },
    RegistrarIndex: 'u32',
    RegistrarInfo: {
      account: 'AccountId',
      fee: 'Balance',
      fields: 'IdentityFields'
    }
  }
};
exports.default = _default;