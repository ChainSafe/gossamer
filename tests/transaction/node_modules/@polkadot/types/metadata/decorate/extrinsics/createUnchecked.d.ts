import type { Registry } from '@polkadot/types-codec/types';
import type { FunctionMetadataLatest } from '../../../interfaces';
import type { CallFunction } from '../../../types';
/** @internal */
export declare function createUnchecked(registry: Registry, section: string, callIndex: Uint8Array, callMetadata: FunctionMetadataLatest): CallFunction;
