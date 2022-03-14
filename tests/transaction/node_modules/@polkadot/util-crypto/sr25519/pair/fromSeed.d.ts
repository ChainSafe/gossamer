import type { HexString } from '@polkadot/util/types';
import type { Keypair } from '../../types';
/**
 * @name sr25519PairFromSeed
 * @description Returns a object containing a `publicKey` & `secretKey` generated from the supplied seed.
 */
export declare function sr25519PairFromSeed(seed: HexString | Uint8Array | string): Keypair;
