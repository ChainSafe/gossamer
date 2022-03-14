import type { Codec } from '@polkadot/types-codec/types';
export type { AnyJson, AnyFunction, AnyNumber, AnyString, AnyTuple, AnyU8a, ArgsDef, BareOpts, Codec, CodecClass, CodecClass as Constructor, CodecTo, Inspect } from '@polkadot/types-codec/types';
export declare type ArrayElementType<T extends ReadonlyArray<unknown>> = T extends ReadonlyArray<infer ElementType> ? ElementType : never;
export declare type Callback<T, E = undefined> = E extends Codec ? (result: T, extra: E) => void | Promise<void> : (result: T) => void | Promise<void>;
