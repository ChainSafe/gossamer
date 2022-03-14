// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { stringToU8a, u8aConcat } from '@polkadot/util';
import { blake2AsU8a } from "../blake2/asU8a.js";
const SS58_PREFIX = stringToU8a('SS58PRE');
export function sshash(key) {
  return blake2AsU8a(u8aConcat(SS58_PREFIX, key), 512);
}