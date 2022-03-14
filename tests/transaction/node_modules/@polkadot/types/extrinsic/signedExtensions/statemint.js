// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
export const statemint = {
  ChargeAssetTxPayment: {
    extrinsic: {
      tip: 'Compact<Balance>',
      // eslint-disable-next-line sort-keys
      assetId: 'Option<AssetId>'
    },
    payload: {}
  }
};