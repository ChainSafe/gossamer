import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataV9, MetadataV10 } from '../../interfaces/metadata';
/** @internal */
export declare function toV10(registry: Registry, { modules }: MetadataV9): MetadataV10;
