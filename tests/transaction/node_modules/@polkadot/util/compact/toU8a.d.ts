/// <reference types="bn.js" />
import { BN } from '../bn';
/**
 * @name compactToU8a
 * @description Encodes a number into a compact representation
 * @example
 * <BR>
 *
 * ```javascript
 * import { compactToU8a } from '@polkadot/util';
 *
 * console.log(compactToU8a(511, 32)); // Uint8Array([0b11111101, 0b00000111])
 * ```
 */
export declare function compactToU8a(value: BN | bigint | number): Uint8Array;
