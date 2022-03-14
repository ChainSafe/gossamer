import type { AnyJson, Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest } from '../../interfaces/metadata';
/** @internal */
export declare function toCallsOnly(registry: Registry, { extrinsic, lookup, pallets }: MetadataLatest): AnyJson;
