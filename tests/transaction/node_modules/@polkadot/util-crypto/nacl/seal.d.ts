interface Sealed {
    sealed: Uint8Array;
    nonce: Uint8Array;
}
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
export declare function naclSeal(message: Uint8Array, senderBoxSecret: Uint8Array, receiverBoxPublic: Uint8Array, nonce?: Uint8Array): Sealed;
export {};
