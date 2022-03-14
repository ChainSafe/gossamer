import type { Enum } from '@polkadot/types-codec';
/** @name StorageKind */
export interface StorageKind extends Enum {
    readonly isPersistent: boolean;
    readonly isLocal: boolean;
    readonly type: 'Persistent' | 'Local';
}
export declare type PHANTOM_OFFCHAIN = 'offchain';
