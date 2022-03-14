/**
 * @name arrayRange
 * @summary Returns a range of numbers ith the size and the specified offset
 * @description
 * Returns a new array of numbers with the specific size. Optionally, when `startAt`, is provided, it generates the range to start at a specific value.
 * @example
 * <BR>
 *
 * ```javascript
 * import { arrayRange } from '@polkadot/util';
 *
 * arrayRange(5); // [0, 1, 2, 3, 4]
 * arrayRange(3, 5); // [5, 6, 7]
 * ```
 */
export declare function arrayRange(size: number, startAt?: number): number[];
