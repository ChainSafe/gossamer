// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { sha256 as sha256Js } from '@noble/hashes/sha256';
import { sha512 as sha512Js } from '@noble/hashes/sha512';
import { sha256, sha512 } from '@polkadot/wasm-crypto';
import { createBitHasher, createDualHasher } from "../helpers.js";
/**
 * @name shaAsU8a
 * @summary Creates a sha Uint8Array from the input.
 */

export const shaAsU8a = createDualHasher({
  256: sha256,
  512: sha512
}, {
  256: sha256Js,
  512: sha512Js
});
/**
 * @name sha256AsU8a
 * @summary Creates a sha256 Uint8Array from the input.
 */

export const sha256AsU8a = createBitHasher(256, shaAsU8a);
/**
 * @name sha512AsU8a
 * @summary Creates a sha512 Uint8Array from the input.
 */

export const sha512AsU8a = createBitHasher(512, shaAsU8a);