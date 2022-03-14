import type { Balance } from '@polkadot/types/interfaces';
export interface DeriveContributions {
    blockHash: string;
    contributorsHex: string[];
}
export declare type DeriveOwnContributions = Record<string, Balance>;
