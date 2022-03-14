import type { AnyU8a, CodecClass, Registry, U8aBitLength } from '../types';
import { Raw } from '../native/Raw';
/**
 * @name U8aFixed
 * @description
 * A U8a that manages a a sequence of bytes up to the specified bitLength. Not meant
 * to be used directly, rather is should be subclassed with the specific lengths.
 */
export declare class U8aFixed extends Raw {
    constructor(registry: Registry, value?: AnyU8a, bitLength?: U8aBitLength);
    static with(bitLength: U8aBitLength, typeName?: string): CodecClass<U8aFixed>;
    /**
     * @description Returns the base runtime type name for this instance
     */
    toRawType(): string;
}
