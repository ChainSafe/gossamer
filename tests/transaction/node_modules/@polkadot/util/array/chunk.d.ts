/**
 * @name arrayChunk
 * @summary Split T[] into T[][] based on the defind size
 * @description
 * Returns a set ao arrays based on the chunksize
 * @example
 * <BR>
 *
 * ```javascript
 * import { arrayChunk } from '@polkadot/util';
 *
 * arrayChunk([1, 2, 3, 4, 5]); // [[1, 2], [3, 4], [5]]
 * ```
 */
export declare function arrayChunk<T>(array: T[], chunkSize: number): T[][];
