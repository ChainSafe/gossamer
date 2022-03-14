import type { ICompact, Inspect, INumber } from '@polkadot/types-codec/types';
import type { StorageEntryMetadataLatest, StorageHasher } from '../../../interfaces/metadata';
import type { StorageEntry } from '../../../primitive/types';
import type { Registry } from '../../../types';
export interface CreateItemOptions {
    key?: Uint8Array | string;
    skipHashing?: boolean;
}
export interface CreateItemBase {
    method: string;
    prefix: string;
}
export interface CreateItemFn extends CreateItemBase {
    meta: StorageEntryMetadataLatest;
    section: string;
}
interface RawArgs {
    args: unknown[];
    hashers: StorageHasher[];
    keys: ICompact<INumber>[];
}
export declare const NO_RAW_ARGS: RawArgs;
/** @internal */
export declare function createKeyRawParts(registry: Registry, itemFn: CreateItemBase, { args, hashers, keys }: RawArgs): [Uint8Array[], Uint8Array[]];
/** @internal */
export declare function createKeyInspect(registry: Registry, itemFn: CreateItemFn, args: RawArgs): Inspect;
/** @internal */
export declare function createKeyRaw(registry: Registry, itemFn: CreateItemBase, args: RawArgs): Uint8Array;
/** @internal */
export declare function createFunction(registry: Registry, itemFn: CreateItemFn, options: CreateItemOptions): StorageEntry;
export {};
