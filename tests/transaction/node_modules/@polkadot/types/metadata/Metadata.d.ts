import type { Registry } from '@polkadot/types-codec/types';
import type { HexString } from '@polkadot/util/types';
import { MetadataVersioned } from './MetadataVersioned';
/**
 * @name Metadata
 * @description
 * The versioned runtime metadata as a decoded structure
 */
export declare class Metadata extends MetadataVersioned {
    constructor(registry: Registry, value?: Uint8Array | HexString | Map<string, unknown> | Record<string, unknown>);
}
