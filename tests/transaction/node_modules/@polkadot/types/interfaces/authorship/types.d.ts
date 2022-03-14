import type { Enum, Option } from '@polkadot/types-codec';
import type { ITuple } from '@polkadot/types-codec/types';
import type { AccountId, BlockNumber, Hash } from '@polkadot/types/interfaces/runtime';
/** @name UncleEntryItem */
export interface UncleEntryItem extends Enum {
    readonly isInclusionHeight: boolean;
    readonly asInclusionHeight: BlockNumber;
    readonly isUncle: boolean;
    readonly asUncle: ITuple<[Hash, Option<AccountId>]>;
    readonly type: 'InclusionHeight' | 'Uncle';
}
export declare type PHANTOM_AUTHORSHIP = 'authorship';
