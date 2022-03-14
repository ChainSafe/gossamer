import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataV12, MetadataV13 } from '../../interfaces/metadata';
/**
 * @internal
 **/
export declare function toV13(registry: Registry, metadata: MetadataV12): MetadataV13;
