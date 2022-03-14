// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import nacl from 'tweetnacl';
/**
 * @name naclBoxPairFromSecret
 * @summary Creates a new public/secret box keypair from a secret.
 * @description
 * Returns a object containing a box `publicKey` & `secretKey` generated from the supplied secret.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclBoxPairFromSecret } from '@polkadot/util-crypto';
 *
 * naclBoxPairFromSecret(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */

export function naclBoxPairFromSecret(secret) {
  return nacl.box.keyPair.fromSecretKey(secret.slice(0, 32));
}