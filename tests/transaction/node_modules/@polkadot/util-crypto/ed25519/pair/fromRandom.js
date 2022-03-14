// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { randomAsU8a } from "../../random/index.js";
import { ed25519PairFromSeed } from "./fromSeed.js";
/**
 * @name ed25519PairFromRandom
 * @summary Creates a new public/secret keypair.
 * @description
 * Returns a new generate object containing a `publicKey` & `secretKey`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromRandom } from '@polkadot/util-crypto';
 *
 * ed25519PairFromRandom(); // => { secretKey: [...], publicKey: [...] }
 * ```
 */

export function ed25519PairFromRandom() {
  return ed25519PairFromSeed(randomAsU8a());
}