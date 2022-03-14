import type { Registry } from '@polkadot/types-codec/types';
import type { StorageEntry } from '../../../primitive/types';
declare type Creator = (registry: Registry) => StorageEntry;
export declare const substrate: Record<string, Creator>;
export {};
