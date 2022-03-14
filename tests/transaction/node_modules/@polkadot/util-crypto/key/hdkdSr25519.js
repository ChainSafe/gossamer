// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { sr25519DeriveHard } from "../sr25519/deriveHard.js";
import { sr25519DeriveSoft } from "../sr25519/deriveSoft.js";
export function keyHdkdSr25519(keypair, {
  chainCode,
  isSoft
}) {
  return isSoft ? sr25519DeriveSoft(keypair, chainCode) : sr25519DeriveHard(keypair, chainCode);
}