// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { blake2AsU8a } from "../blake2/index.js";
import { keccakAsU8a } from "../keccak/index.js";
export function hasher(hashType, data, onlyJs) {
  return hashType === 'keccak' ? keccakAsU8a(data, undefined, onlyJs) : blake2AsU8a(data, undefined, undefined, onlyJs);
}