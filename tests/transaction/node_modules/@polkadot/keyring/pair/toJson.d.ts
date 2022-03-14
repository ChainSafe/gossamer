import type { KeypairType } from '@polkadot/util-crypto/types';
import type { KeyringPair$Json, KeyringPair$Meta } from '../types';
interface PairStateJson {
    address: string;
    meta: KeyringPair$Meta;
}
export declare function pairToJson(type: KeypairType, { address, meta }: PairStateJson, encoded: Uint8Array, isEncrypted: boolean): KeyringPair$Json;
export {};
