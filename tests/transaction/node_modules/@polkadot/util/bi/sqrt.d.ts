/// <reference types="bn.js" />
import type { BN } from '../bn';
import type { ToBigInt, ToBn } from '../types';
/**
 * @name nSqrt
 * @summary Calculates the integer square root of a bigint
 */
export declare function nSqrt<ExtToBn extends ToBn | ToBigInt>(value: ExtToBn | BN | bigint | string | number | null): bigint;
