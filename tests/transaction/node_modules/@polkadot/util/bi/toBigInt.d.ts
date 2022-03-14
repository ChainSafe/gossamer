/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { HexString, ToBigInt, ToBn } from '../types';
/**
 * @name nToBigInt
 * @summary Creates a bigInt value from a BN, bigint, string (base 10 or hex) or number input.
 */
export declare function nToBigInt<ExtToBn extends ToBigInt | ToBn>(value?: HexString | ExtToBn | BN | bigint | string | number | null): bigint;
