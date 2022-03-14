import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest } from '../../../interfaces';
import type { Storage } from '../types';
/** @internal */
export declare function decorateStorage(registry: Registry, { pallets }: MetadataLatest, _metaVersion: number): Storage;
