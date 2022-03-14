// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aConcat, u8aToU8a } from '@polkadot/util';
import { blake2AsU8a, xxhashAsU8a } from '@polkadot/util-crypto';

const DEFAULT_FN = data => xxhashAsU8a(data, 128);

const HASHERS = {
  Blake2_128: data => // eslint-disable-line camelcase
  blake2AsU8a(data, 128),
  Blake2_128Concat: data => // eslint-disable-line camelcase
  u8aConcat(blake2AsU8a(data, 128), u8aToU8a(data)),
  Blake2_256: data => // eslint-disable-line camelcase
  blake2AsU8a(data, 256),
  Identity: data => u8aToU8a(data),
  Twox128: data => xxhashAsU8a(data, 128),
  Twox256: data => xxhashAsU8a(data, 256),
  Twox64Concat: data => u8aConcat(xxhashAsU8a(data, 64), u8aToU8a(data))
};
/** @internal */

export function getHasher(hasher) {
  return HASHERS[hasher.type] || DEFAULT_FN;
}