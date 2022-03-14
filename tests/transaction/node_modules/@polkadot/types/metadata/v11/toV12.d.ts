import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataV11, MetadataV12 } from '../../interfaces/metadata';
/**
 * @internal
 **/
export declare function toV12(registry: Registry, { extrinsic, modules }: MetadataV11): MetadataV12;
