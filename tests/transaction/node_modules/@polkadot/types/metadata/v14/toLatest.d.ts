import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest, MetadataV14 } from '../../interfaces/metadata';
/**
 * Convert the Metadata (which is an alias) to latest
 * @internal
 **/
export declare function toLatest(registry: Registry, v14: MetadataV14, _metaVersion: number): MetadataLatest;
