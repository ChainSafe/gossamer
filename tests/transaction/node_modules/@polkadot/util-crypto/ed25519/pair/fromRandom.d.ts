import type { Keypair } from '../../types';
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
export declare function ed25519PairFromRandom(): Keypair;
