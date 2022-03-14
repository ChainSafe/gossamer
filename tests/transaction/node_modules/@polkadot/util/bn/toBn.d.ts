/// <reference types="bn.js" />
import type { HexString, ToBigInt, ToBn } from '../types';
import { BN } from './bn';
/**
 * @name bnToBn
 * @summary Creates a BN value from a BN, bigint, string (base 10 or hex) or number input.
 * @description
 * `null` inputs returns a `0x0` result, BN values returns the value, numbers returns a BN representation.
 * @example
 * <BR>
 *
 * ```javascript
 * import BN from 'bn.js';
 * import { bnToBn } from '@polkadot/util';
 *
 * bnToBn(0x1234); // => BN(0x1234)
 * bnToBn(new BN(0x1234)); // => BN(0x1234)
 * ```
 */
export declare function bnToBn<ExtToBn extends ToBigInt | ToBn>(value?: HexString | ExtToBn | BN | bigint | string | number | null): BN;
