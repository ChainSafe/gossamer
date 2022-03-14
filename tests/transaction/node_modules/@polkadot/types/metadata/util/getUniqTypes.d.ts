import type { Registry } from '@polkadot/types-codec/types';
import type { MetadataLatest } from '../../interfaces';
/** @internal */
export declare function getUniqTypes(registry: Registry, meta: MetadataLatest, throwError: boolean): string[];
