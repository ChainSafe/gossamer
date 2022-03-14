import type { HexString } from '@polkadot/util/types';
/**
 * @name addressEq
 * @summary Compares two addresses, either in ss58, Uint8Array or hex format.
 * @description
 * For the input values, return true is the underlying public keys do match.
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aEq } from '@polkadot/util';
 *
 * u8aEq(new Uint8Array([0x68, 0x65]), new Uint8Array([0x68, 0x65])); // true
 * ```
 */
export declare function addressEq(a: HexString | string | Uint8Array, b: HexString | string | Uint8Array): boolean;
