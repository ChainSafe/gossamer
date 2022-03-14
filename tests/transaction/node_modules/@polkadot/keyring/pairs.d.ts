import type { HexString } from '@polkadot/util/types';
import type { KeyringPair, KeyringPairs } from './types';
export declare class Pairs implements KeyringPairs {
    #private;
    add(pair: KeyringPair): KeyringPair;
    all(): KeyringPair[];
    get(address: HexString | string | Uint8Array): KeyringPair;
    remove(address: HexString | string | Uint8Array): void;
}
