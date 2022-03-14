import type { AccountId, Header } from '@polkadot/types/interfaces';
import type { Registry } from '@polkadot/types/types';
import type { HeaderExtended } from './types';
export declare function createHeaderExtended(registry: Registry, header?: Header, validators?: AccountId[]): HeaderExtended;
