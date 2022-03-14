import type { Keypair } from '../../types';
/**
 * @name secp256k1PairFromSeed
 * @description Returns a object containing a `publicKey` & `secretKey` generated from the supplied seed.
 */
export declare function secp256k1PairFromSeed(seed: Uint8Array, onlyJs?: boolean): Keypair;
