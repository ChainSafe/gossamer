import type { StorageEntry } from '@polkadot/types/primitive/types';
import type { Registry } from '@polkadot/types/types';
export declare function extractStorageArgs(registry: Registry, creator: StorageEntry, _args: unknown[]): [StorageEntry, unknown[]];
