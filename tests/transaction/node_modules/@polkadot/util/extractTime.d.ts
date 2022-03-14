import type { Time } from './types';
/**
 * @name extractTime
 * @summary Convert a quantity of seconds to Time array representing accumulated {days, minutes, hours, seconds, milliseconds}
 * @example
 * <BR>
 *
 * ```javascript
 * import { extractTime } from '@polkadot/util';
 *
 * const { days, minutes, hours, seconds, milliseconds } = extractTime(6000); // 0, 0, 10, 0, 0
 * ```
 */
export declare function extractTime(milliseconds?: number): Time;
