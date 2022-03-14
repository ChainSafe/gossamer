// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { bnToU8a, stringToU8a, u8aConcat } from '@polkadot/util';
import { blake2AsU8a } from "../blake2/asU8a.js";
import { BN_LE_16_OPTS } from "../bn.js";
import { decodeAddress } from "./decode.js";
const PREFIX = stringToU8a('modlpy/utilisuba');
export function createKeyDerived(who, index) {
  return blake2AsU8a(u8aConcat(PREFIX, decodeAddress(who), bnToU8a(index, BN_LE_16_OPTS)));
}