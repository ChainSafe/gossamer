// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
// order important in structs... :)

/* eslint-disable sort-keys */
export default {
  rpc: {},
  types: {
    UncleEntryItem: {
      _enum: {
        InclusionHeight: 'BlockNumber',
        Uncle: '(Hash, Option<AccountId>)'
      }
    }
  }
};