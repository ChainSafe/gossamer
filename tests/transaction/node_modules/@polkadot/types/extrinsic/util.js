// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
// a helper function for both types of payloads, Raw and metadata-known
export function sign(registry, signerPair, u8a, options) {
  const encoded = u8a.length > 256 ? registry.hash(u8a) : u8a;
  return signerPair.sign(encoded, options);
}