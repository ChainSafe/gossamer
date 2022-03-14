/// <reference types="node" />
import type { StorageHasher } from '../../../interfaces';
export declare type HasherInput = string | Buffer | Uint8Array;
export declare type HasherFunction = (data: HasherInput) => Uint8Array;
/** @internal */
export declare function getHasher(hasher: StorageHasher): HasherFunction;
