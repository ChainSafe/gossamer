import type { HexString } from '@polkadot/util/types';
import type { HashType } from './types';
/**
 * @name secp256k1Recover
 * @description Recovers a publicKey from the supplied signature
 */
export declare function secp256k1Recover(msgHash: HexString | Uint8Array | string, signature: HexString | Uint8Array | string, recovery: number, hashType?: HashType, onlyJs?: boolean): Uint8Array;
