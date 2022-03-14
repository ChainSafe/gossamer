import type { HexString } from '@polkadot/util/types';
/**
 * @name ed25519Sign
 * @summary Verifies the signature on the supplied message.
 * @description
 * Verifies the `signature` on `message` with the supplied `publicKey`. Returns `true` on sucess, `false` otherwise.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519Verify } from '@polkadot/util-crypto';
 *
 * ed25519Verify([...], [...], [...]); // => true/false
 * ```
 */
export declare function ed25519Verify(message: HexString | Uint8Array | string, signature: HexString | Uint8Array | string, publicKey: HexString | Uint8Array | string, onlyJs?: boolean): boolean;
