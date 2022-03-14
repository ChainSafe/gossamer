// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { emptyCheck } from "./emptyCheck.js";
const CheckMortality = {
  extrinsic: {
    era: 'ExtrinsicEra'
  },
  payload: {
    blockHash: 'Hash'
  }
};
export const substrate = {
  ChargeTransactionPayment: {
    extrinsic: {
      tip: 'Compact<Balance>'
    },
    payload: {}
  },
  CheckBlockGasLimit: emptyCheck,
  CheckEra: CheckMortality,
  CheckGenesis: {
    extrinsic: {},
    payload: {
      genesisHash: 'Hash'
    }
  },
  CheckMortality,
  CheckNonZeroSender: emptyCheck,
  CheckNonce: {
    extrinsic: {
      nonce: 'Compact<Index>'
    },
    payload: {}
  },
  CheckSpecVersion: {
    extrinsic: {},
    payload: {
      specVersion: 'u32'
    }
  },
  CheckTxVersion: {
    extrinsic: {},
    payload: {
      transactionVersion: 'u32'
    }
  },
  CheckVersion: {
    extrinsic: {},
    payload: {
      specVersion: 'u32'
    }
  },
  CheckWeight: emptyCheck,
  LockStakingStatus: emptyCheck,
  ValidateEquivocationReport: emptyCheck
};