import type { Struct, bool, u8 } from '@polkadot/types-codec';
import type { ITuple } from '@polkadot/types-codec/types';
import type { Balance, BlockNumber } from '@polkadot/types/interfaces/runtime';
/** @name CallIndex */
export interface CallIndex extends ITuple<[u8, u8]> {
}
/** @name LotteryConfig */
export interface LotteryConfig extends Struct {
    readonly price: Balance;
    readonly start: BlockNumber;
    readonly length: BlockNumber;
    readonly delay: BlockNumber;
    readonly repeat: bool;
}
export declare type PHANTOM_LOTTERY = 'lottery';
