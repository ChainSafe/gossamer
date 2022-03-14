/// <reference types="bn.js" />
import type { BN } from '../bn/bn';
import type { HexString, ToBigInt, ToBn, ToBnOptions } from '../types';
interface Options extends ToBnOptions {
    bitLength?: number;
}
/**
 * @name nToHex
 * @summary Creates a hex value from a bigint object.
 */
export declare function nToHex<ExtToBn extends ToBn | ToBigInt>(value?: ExtToBn | BN | bigint | number | null, options?: Options): HexString;
export {};
