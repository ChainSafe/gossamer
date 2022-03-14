// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { pbkdf2 as pbkdf2Js } from '@noble/hashes/pbkdf2';
import { sha512 } from '@noble/hashes/sha512';
import { hasBigInt, u8aToU8a } from '@polkadot/util';
import { isReady, pbkdf2 } from '@polkadot/wasm-crypto';
import { randomAsU8a } from "../random/asU8a.js";
export function pbkdf2Encode(passphrase, salt = randomAsU8a(), rounds = 2048, onlyJs) {
  const u8aPass = u8aToU8a(passphrase);
  const u8aSalt = u8aToU8a(salt);
  return {
    password: !hasBigInt || !onlyJs && isReady() ? pbkdf2(u8aPass, u8aSalt, rounds) : pbkdf2Js(sha512, u8aPass, u8aSalt, {
      c: rounds,
      dkLen: 64
    }),
    rounds,
    salt
  };
}