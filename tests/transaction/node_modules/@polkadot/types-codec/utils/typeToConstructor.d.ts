import type { Codec, CodecClass, Registry } from '../types';
export declare function typeToConstructor<T extends Codec = Codec>(registry: Registry, type: string | CodecClass<T>): CodecClass<T>;
