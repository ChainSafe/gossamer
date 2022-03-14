import type { AnyString } from '../types';
/**
 * @name stringShorten
 * @summary Returns a string with maximum length
 * @description
 * Checks the string against the `prefixLength`, if longer than double this, shortens it by placing `..` in the middle of it
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringShorten } from '@polkadot/util';
 *
 * stringShorten('1234567890', 2); // => 12..90
 * ```
 */
export declare function stringShorten(value: AnyString, prefixLength?: number): string;
