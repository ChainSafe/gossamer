import type { Inspect, Registry } from '@polkadot/types-codec/types';
import { Enum } from '@polkadot/types-codec';
export declare class GenericMultiAddress extends Enum {
    constructor(registry: Registry, value?: unknown);
    /**
     * @description Returns a breakdown of the hex encoding for this Codec
     */
    inspect(): Inspect;
    /**
     * @description Returns the string representation of the value
     */
    toString(): string;
}
