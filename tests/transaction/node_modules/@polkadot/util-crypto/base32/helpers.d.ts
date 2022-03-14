import type { U8aLike } from '@polkadot/util/types';
export type { U8aLike } from '@polkadot/util/types';
interface Coder {
    decode: (value: string) => Uint8Array;
    encode: (value: Uint8Array) => string;
}
interface Config {
    chars: string;
    coder: Coder;
    ipfs?: string;
    regex?: RegExp;
    type: string;
}
declare type DecodeFn = (value: string, ipfsCompat?: boolean) => Uint8Array;
declare type EncodeFn = (value: U8aLike, ipfsCompat?: boolean) => string;
declare type ValidateFn = (value?: unknown, ipfsCompat?: boolean) => value is string;
export declare function createDecode({ coder, ipfs }: Config, validate: ValidateFn): DecodeFn;
export declare function createEncode({ coder, ipfs }: Config): EncodeFn;
export declare function createIs(validate: ValidateFn): ValidateFn;
export declare function createValidate({ chars, ipfs, type }: Config): ValidateFn;
