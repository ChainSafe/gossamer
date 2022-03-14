import type { ExactDerive } from '@polkadot/api-derive';
import type { AnyFunction } from '@polkadot/types/types';
import type { ApiTypes, DecorateMethod, MethodResult } from '../types';
declare type AnyDeriveSection = Record<string, AnyFunction>;
declare type AnyDerive = Record<string, AnyDeriveSection>;
declare type DeriveSection<ApiType extends ApiTypes, Section extends AnyDeriveSection> = {
    [M in keyof Section]: MethodResult<ApiType, Section[M]>;
};
export declare type AllDerives<ApiType extends ApiTypes> = {
    [S in keyof ExactDerive]: DeriveSection<ApiType, ExactDerive[S]>;
};
/**
 * This is a section decorator which keeps all type information.
 */
export declare function decorateDeriveSections<ApiType extends ApiTypes>(decorateMethod: DecorateMethod<ApiType>, derives: AnyDerive): AllDerives<ApiType>;
export {};
