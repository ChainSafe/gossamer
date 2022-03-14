// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import nacl from 'tweetnacl';
/**
 * @name naclOpen
 * @summary Opens a message using the receiver's secretKey and nonce
 * @description
 * Returns a message sealed by the sender, using the receiver's `secret` and `nonce`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclOpen } from '@polkadot/util-crypto';
 *
 * naclOpen([...], [...], [...]); // => [...]
 * ```
 */

export function naclOpen(sealed, nonce, senderBoxPublic, receiverBoxSecret) {
  return nacl.box.open(sealed, nonce, senderBoxPublic, receiverBoxSecret) || null;
}