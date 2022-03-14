import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest } from '../../../interfaces';
import type { Constants } from '../types';
/** @internal */
export declare function decorateConstants(registry: Registry, { pallets }: MetadataLatest, _version: number): Constants;
