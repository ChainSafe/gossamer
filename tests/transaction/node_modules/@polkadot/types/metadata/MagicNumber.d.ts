import type { AnyNumber, Registry } from '@polkadot/types-codec/types';
import { U32 } from '@polkadot/types-codec';
export declare const MAGIC_NUMBER = 1635018093;
export declare class MagicNumber extends U32 {
    constructor(registry: Registry, value?: AnyNumber);
}
