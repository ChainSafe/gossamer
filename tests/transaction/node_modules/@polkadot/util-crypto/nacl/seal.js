// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import nacl from 'tweetnacl';
import { randomAsU8a } from "../random/asU8a.js";

/**
 * @name naclSeal
 * @summary Seals a message using the sender's encrypting secretKey, receiver's public key, and nonce
 * @description
 * Returns an encrypted message which can be open only by receiver's secretKey. If the `nonce` was not supplied, a random value is generated.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclSeal } from '@polkadot/util-crypto';
 *
 * naclSeal([...], [...], [...], [...]); // => [...]
 * ```
 */
export function naclSeal(message, senderBoxSecret, receiverBoxPublic, nonce = randomAsU8a(24)) {
  return {
    nonce,
    sealed: nacl.box(message, nonce, receiverBoxPublic, senderBoxSecret)
  };
}