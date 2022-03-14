import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataV10, MetadataV11 } from '../../interfaces/metadata';
/** @internal */
export declare function toV11(registry: Registry, { modules }: MetadataV10): MetadataV11;
