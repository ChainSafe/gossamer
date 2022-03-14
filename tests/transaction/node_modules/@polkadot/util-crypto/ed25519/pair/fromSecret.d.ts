import type { Keypair } from '../../types';
/**
 * @name ed25519PairFromSecret
 * @summary Creates a new public/secret keypair from a secret.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied secret.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromSecret } from '@polkadot/util-crypto';
 *
 * ed25519PairFromSecret(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
export declare function ed25519PairFromSecret(secret: Uint8Array): Keypair;
