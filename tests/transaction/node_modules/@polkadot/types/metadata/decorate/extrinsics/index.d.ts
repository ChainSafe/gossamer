import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest, PalletMetadataLatest, SiVariant } from '../../../interfaces';
import type { PortableRegistry } from '../../../metadata';
import type { CallFunction } from '../../../types';
import type { Extrinsics } from '../types';
export declare function filterCallsSome({ calls }: PalletMetadataLatest): boolean;
export declare function createCallFunction(registry: Registry, lookup: PortableRegistry, variant: SiVariant, sectionName: string, sectionIndex: number): CallFunction;
/** @internal */
export declare function decorateExtrinsics(registry: Registry, { lookup, pallets }: MetadataLatest, version: number): Extrinsics;
