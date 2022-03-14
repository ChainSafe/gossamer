// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { stringToU8a } from '@polkadot/util';
import { blake2AsU8a } from "../../blake2/asU8a.js";
import { ed25519PairFromSeed } from "./fromSeed.js";
/**
 * @name ed25519PairFromString
 * @summary Creates a new public/secret keypair from a string.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied string. The string is hashed and the value used as the input seed.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromString } from '@polkadot/util-crypto';
 *
 * ed25519PairFromString('test'); // => { secretKey: [...], publicKey: [...] }
 * ```
 */

export function ed25519PairFromString(value) {
  return ed25519PairFromSeed(blake2AsU8a(stringToU8a(value)));
}