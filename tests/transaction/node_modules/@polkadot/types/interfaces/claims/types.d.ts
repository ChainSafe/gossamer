import type { Enum } from '@polkadot/types-codec';
/** @name StatementKind */
export interface StatementKind extends Enum {
    readonly isRegular: boolean;
    readonly isSaft: boolean;
    readonly type: 'Regular' | 'Saft';
}
export declare type PHANTOM_CLAIMS = 'claims';
