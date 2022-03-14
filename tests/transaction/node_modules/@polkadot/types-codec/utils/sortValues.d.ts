/// <reference types="bn.js" />
import type { BN } from '@polkadot/util';
import type { Codec } from '../types';
declare type SortArg = Codec | Codec[] | number[] | BN | bigint | number | Uint8Array;
/**
* Sort keys/values of BTreeSet/BTreeMap in ascending order for encoding compatibility with Rust's BTreeSet/BTreeMap
* (https://doc.rust-lang.org/stable/std/collections/struct.BTreeSet.html)
* (https://doc.rust-lang.org/stable/std/collections/struct.BTreeMap.html)
*/
export declare function sortAsc<V extends SortArg = Codec>(a: V, b: V): number;
export declare function sortSet<V extends Codec = Codec>(set: Set<V>): Set<V>;
export declare function sortMap<K extends Codec = Codec, V extends Codec = Codec>(map: Map<K, V>): Map<K, V>;
export {};
