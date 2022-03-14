import type { KeyringInstance, KeyringOptions } from './types';
import { KeypairType } from '@polkadot/util-crypto/types';
interface PairDef {
    name?: string;
    publicKey: Uint8Array;
    seed?: string;
    secretKey?: Uint8Array;
    type: KeypairType;
}
export declare const PAIRSSR25519: PairDef[];
export declare const PAIRSETHEREUM: PairDef[];
/**
 * @name testKeyring
 * @summary Create an instance of Keyring pre-populated with locked test accounts
 * @description The test accounts (i.e. alice, bob, dave, eve, ferdie)
 * are available on the dev chain and each test account is initialized with DOT funds.
 */
export declare function createTestKeyring(options?: KeyringOptions, isDerived?: boolean): KeyringInstance;
export {};
