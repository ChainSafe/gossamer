import type { Keypair } from '../../types';
/**
 * @name ed25519PairFromString
 * @summary Creates a new public/secret keypair from a string.
 * @description
 * Returns a object containing a `publicKey` & `secretKey` generated from the supplied string. The string is hashed and the value used as the input seed.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519PairFromString } from '@polkadot/util-crypto';
 *
 * ed25519PairFromString('test'); // => { secretKey: [...], publicKey: [...] }
 * ```
 */
export declare function ed25519PairFromString(value: string): Keypair;
