/// <reference types="bn.js" />
import type { ToBn } from '../types';
import { BN } from './bn';
/**
 * @name bnSqrt
 * @summary Calculates the integer square root of a BN
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnSqrt } from '@polkadot/util';
 *
 * bnSqrt(new BN(16)).toString(); // => '4'
 * ```
 */
export declare function bnSqrt<ExtToBn extends ToBn>(value: ExtToBn | BN | bigint | string | number | null): BN;
