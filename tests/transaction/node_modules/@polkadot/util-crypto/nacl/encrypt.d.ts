interface Encrypted {
    encrypted: Uint8Array;
    nonce: Uint8Array;
}
/**
 * @name naclEncrypt
 * @summary Encrypts a message using the supplied secretKey and nonce
 * @description
 * Returns an encrypted message, using the `secretKey` and `nonce`. If the `nonce` was not supplied, a random value is generated.
 * @example
 * <BR>
 *
 * ```javascript
 * import { naclEncrypt } from '@polkadot/util-crypto';
 *
 * naclEncrypt([...], [...]); // => [...]
 * ```
 */
export declare function naclEncrypt(message: Uint8Array, secret: Uint8Array, nonce?: Uint8Array): Encrypted;
export {};
