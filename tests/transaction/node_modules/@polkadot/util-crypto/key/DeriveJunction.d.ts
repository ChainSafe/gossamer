/// <reference types="bn.js" />
import type { HexString } from '@polkadot/util/types';
import { BN } from '@polkadot/util';
export declare class DeriveJunction {
    #private;
    static from(value: string): DeriveJunction;
    get chainCode(): Uint8Array;
    get isHard(): boolean;
    get isSoft(): boolean;
    hard(value: HexString | number | string | bigint | BN | Uint8Array): DeriveJunction;
    harden(): DeriveJunction;
    soft(value: HexString | number | string | bigint | BN | Uint8Array): DeriveJunction;
    soften(): DeriveJunction;
}
