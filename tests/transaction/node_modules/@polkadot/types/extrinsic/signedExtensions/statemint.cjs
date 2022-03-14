"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.statemint = void 0;
// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
const statemint = {
  ChargeAssetTxPayment: {
    extrinsic: {
      tip: 'Compact<Balance>',
      // eslint-disable-next-line sort-keys
      assetId: 'Option<AssetId>'
    },
    payload: {}
  }
};
exports.statemint = statemint;