// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { bnToU8a, u8aConcat } from '@polkadot/util';
import { BN_LE_32_OPTS } from "../bn.js";
export function scryptToU8a(salt, {
  N,
  p,
  r
}) {
  return u8aConcat(salt, bnToU8a(N, BN_LE_32_OPTS), bnToU8a(p, BN_LE_32_OPTS), bnToU8a(r, BN_LE_32_OPTS));
}