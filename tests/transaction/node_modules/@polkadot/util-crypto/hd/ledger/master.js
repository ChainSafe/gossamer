// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aConcat } from '@polkadot/util';
import { hmacShaAsU8a } from "../../hmac/index.js";
import { mnemonicToSeedSync } from "../../mnemonic/bip39.js";
const ED25519_CRYPTO = 'ed25519 seed'; // gets an xprv from a mnemonic

export function ledgerMaster(mnemonic, password) {
  const seed = mnemonicToSeedSync(mnemonic, password);
  const chainCode = hmacShaAsU8a(ED25519_CRYPTO, new Uint8Array([1, ...seed]), 256);
  let priv;

  while (!priv || priv[31] & 0b00100000) {
    priv = hmacShaAsU8a(ED25519_CRYPTO, priv || seed, 512);
  }

  priv[0] &= 0b11111000;
  priv[31] &= 0b01111111;
  priv[31] |= 0b01000000;
  return u8aConcat(priv, chainCode);
}