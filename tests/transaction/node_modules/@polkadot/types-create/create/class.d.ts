import type { Codec, CodecClass, Registry } from '@polkadot/types-codec/types';
import type { TypeDef } from '../types';
export declare function constructTypeClass<T extends Codec = Codec>(registry: Registry, typeDef: TypeDef): CodecClass<T>;
export declare function getTypeClass<T extends Codec = Codec>(registry: Registry, typeDef: TypeDef): CodecClass<T>;
export declare function createClassUnsafe<T extends Codec = Codec, K extends string = string>(registry: Registry, type: K): CodecClass<T>;
