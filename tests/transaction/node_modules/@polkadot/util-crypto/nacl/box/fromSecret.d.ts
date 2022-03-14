import type { Keypair } from '../../types';
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
export declare function naclBoxPairFromSecret(secret: Uint8Array): Keypair;
