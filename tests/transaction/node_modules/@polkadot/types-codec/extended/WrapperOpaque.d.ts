import type { Codec, CodecClass, Registry } from '../types';
import { WrapperKeepOpaque } from './WrapperKeepOpaque';
export declare class WrapperOpaque<T extends Codec> extends WrapperKeepOpaque<T> {
    constructor(registry: Registry, typeName: CodecClass<T> | string, value?: unknown);
    static with<T extends Codec>(Type: CodecClass<T> | string): CodecClass<WrapperKeepOpaque<T>>;
    /**
     * @description The inner value for this wrapper, in all cases it _should_ be decodable (unlike KeepOpaque)
     */
    get inner(): T;
}
