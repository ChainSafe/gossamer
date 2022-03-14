// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { hasBigInt } from '@polkadot/util';
import { bip39ToEntropy, isReady } from '@polkadot/wasm-crypto';
import { mnemonicToEntropy as jsToEntropy } from "./bip39.js";
export function mnemonicToEntropy(mnemonic, onlyJs) {
  return !hasBigInt || !onlyJs && isReady() ? bip39ToEntropy(mnemonic) : jsToEntropy(mnemonic);
}