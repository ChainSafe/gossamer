import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest, PalletMetadataLatest } from '../../../interfaces';
import type { Events } from '../types';
export declare function filterEventsSome({ events }: PalletMetadataLatest): boolean;
/** @internal */
export declare function decorateEvents(registry: Registry, { lookup, pallets }: MetadataLatest, version: number): Events;
