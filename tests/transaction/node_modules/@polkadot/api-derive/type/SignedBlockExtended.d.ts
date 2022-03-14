import type { AccountId, EventRecord, SignedBlock } from '@polkadot/types/interfaces';
import type { Registry } from '@polkadot/types/types';
import type { SignedBlockExtended } from './types';
export declare function createSignedBlockExtended(registry: Registry, block?: SignedBlock, events?: EventRecord[], validators?: AccountId[]): SignedBlockExtended;
