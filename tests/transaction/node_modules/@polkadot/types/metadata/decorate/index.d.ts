import type { Registry } from '@polkadot/types-codec/types';
import type { DecoratedMeta } from './types';
import { Metadata } from '../Metadata';
import { decorateConstants } from './constants';
import { decorateErrors } from './errors';
import { decorateEvents, filterEventsSome } from './events';
import { decorateExtrinsics, filterCallsSome } from './extrinsics';
import { decorateStorage } from './storage';
/**
 * Expands the metadata by decoration into consts, query and tx sections
 */
export declare function expandMetadata(registry: Registry, metadata: Metadata): DecoratedMeta;
export { decorateConstants, decorateErrors, decorateEvents, decorateExtrinsics, decorateStorage, filterCallsSome, filterEventsSome };
