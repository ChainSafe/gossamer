import type { Keypair } from '../../types';
/**
 * @name ed25519PairFromSeed
 * @summary Creates a new public/secret keypair from a seed.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied seed.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromSeed } from '@polkadot/util-crypto';
 *
 * ed25519PairFromSeed(...); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
export declare function ed25519PairFromSeed(seed: Uint8Array, onlyJs?: boolean): Keypair;
