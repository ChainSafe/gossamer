import type { HexString } from '@polkadot/util/types';
/**
 * @name sr25519Agreement
 * @description Key agreement between other's public key and self secret key
 */
export declare function sr25519Agreement(secretKey: HexString | Uint8Array | string, publicKey: HexString | Uint8Array | string): Uint8Array;
