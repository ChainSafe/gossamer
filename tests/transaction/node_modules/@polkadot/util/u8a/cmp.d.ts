import type { HexString } from '../types';
/**
 * @name u8aCmp
 * @summary Compares two Uint8Arrays for sorting.
 * @description
 * For `UInt8Array` (or hex string) input values returning -1, 0 or +1
 * @example
 * <BR>
 *
 * ```javascript
 * import { u8aCmp } from '@polkadot/util';
 *
 * u8aCmp(new Uint8Array([0x67, 0x65]), new Uint8Array([0x68, 0x65])); // -1
 * u8aCmp(new Uint8Array([0x68, 0x65]), new Uint8Array([0x68, 0x65])); // 0
 * u8aCmp(new Uint8Array([0x69, 0x65]), new Uint8Array([0x68, 0x65])); // +1
 * ```
 */
export declare function u8aCmp(a: HexString | Uint8Array | string, b: HexString | Uint8Array | string): number;
