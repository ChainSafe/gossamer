import type { RegistryEntry } from '@substrate/ss58-registry';
export declare type Icon = 'beachball' | 'empty' | 'jdenticon' | 'polkadot' | 'substrate';
export declare type KnownIcon = Record<string, Icon>;
export declare type KnownLedger = Record<string, number>;
export declare type KnownGenesis = Record<string, string[]>;
export declare type KnownSubstrate = RegistryEntry;
export declare type KnownTestnet = Record<string, true>;
export interface SubstrateNetwork extends KnownSubstrate {
    genesisHash: string[];
    hasLedgerSupport: boolean;
    icon: Icon;
    isIgnored: boolean;
    isTestnet: boolean;
    slip44?: number | null;
}
export interface Network extends SubstrateNetwork {
    network: string;
}
export interface Ss58Registry {
    registry: KnownSubstrate[];
    specification: string;
    schema: Record<keyof KnownSubstrate, string>;
}
