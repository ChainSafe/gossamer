import type { HexString } from '@polkadot/util/types';
import type { Keypair } from '../types';
/**
 * @name sr25519Sign
 * @description Returns message signature of `message`, using the supplied pair
 */
export declare function sr25519Sign(message: HexString | Uint8Array | string, { publicKey, secretKey }: Partial<Keypair>): Uint8Array;
