/// <reference types="bn.js" />
import type { ToBn, ToBnOptions } from '../types';
import type { BN } from './bn';
interface Options extends ToBnOptions {
    bitLength?: number;
}
/**
 * @name bnToU8a
 * @summary Creates a Uint8Array object from a BN.
 * @description
 * `null`/`undefined`/`NaN` inputs returns an empty `Uint8Array` result. `BN` input values return the actual bytes value converted to a `Uint8Array`. Optionally convert using little-endian format if `isLE` is set.
 * @example
 * <BR>
 *
 * ```javascript
 * import { bnToU8a } from '@polkadot/util';
 *
 * bnToU8a(new BN(0x1234)); // => [0x12, 0x34]
 * ```
 */
declare function bnToU8a<ExtToBn extends ToBn>(value?: ExtToBn | BN | bigint | number | null, options?: Options): Uint8Array;
declare function bnToU8a<ExtToBn extends ToBn>(value?: ExtToBn | BN | bigint | number | null, bitLength?: number, isLe?: boolean): Uint8Array;
export { bnToU8a };
