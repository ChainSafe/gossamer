import type { AnyJson, AnyTuple } from '@polkadot/types-codec/types';
import type { StorageEntryMetadataLatest, StorageEntryTypeLatest } from '../interfaces/metadata';
import type { SiLookupTypeId } from '../interfaces/scaleInfo';
import type { InterfaceTypes, IStorageKey, Registry } from '../types';
import type { StorageEntry } from './types';
import { Bytes } from '@polkadot/types-codec';
interface StorageKeyExtra {
    method: string;
    section: string;
}
export declare function unwrapStorageSi(type: StorageEntryTypeLatest): SiLookupTypeId;
/** @internal */
export declare function unwrapStorageType(registry: Registry, type: StorageEntryTypeLatest, isOptional?: boolean): keyof InterfaceTypes;
/**
 * @name StorageKey
 * @description
 * A representation of a storage key (typically hashed) in the system. It can be
 * constructed by passing in a raw key or a StorageEntry with (optional) arguments.
 */
export declare class StorageKey<A extends AnyTuple = AnyTuple> extends Bytes implements IStorageKey<A> {
    #private;
    constructor(registry: Registry, value?: string | Uint8Array | StorageKey | StorageEntry | [StorageEntry, unknown[]?], override?: Partial<StorageKeyExtra>);
    /**
     * @description Return the decoded arguments (applicable to map with decodable values)
     */
    get args(): A;
    /**
     * @description The metadata or `undefined` when not available
     */
    get meta(): StorageEntryMetadataLatest | undefined;
    /**
     * @description The key method or `undefined` when not specified
     */
    get method(): string | undefined;
    /**
     * @description The output type
     */
    get outputType(): string;
    /**
     * @description The key section or `undefined` when not specified
     */
    get section(): string | undefined;
    is(key: IStorageKey<AnyTuple>): key is IStorageKey<A>;
    /**
     * @description Sets the meta for this key
     */
    setMeta(meta?: StorageEntryMetadataLatest, section?: string, method?: string): this;
    /**
     * @description Returns the Human representation for this type
     */
    toHuman(): AnyJson;
    /**
     * @description Returns the raw type for this
     */
    toRawType(): string;
}
export {};
