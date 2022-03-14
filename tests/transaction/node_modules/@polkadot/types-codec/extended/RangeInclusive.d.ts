import type { AnyTuple, CodecClass, INumber, Registry } from '../types';
import { Range } from './Range';
export declare class RangeInclusive<T extends INumber = INumber> extends Range<T> {
    constructor(registry: Registry, Type: CodecClass<T> | string, value?: AnyTuple);
    static with<T extends INumber>(Type: CodecClass<T> | string): CodecClass<RangeInclusive<T>>;
}
