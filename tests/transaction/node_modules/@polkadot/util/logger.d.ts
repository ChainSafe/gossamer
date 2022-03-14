import type { Logger } from './types';
export declare function loggerFormat(value: unknown): unknown;
/**
 * @name Logger
 * @summary Creates a consistent log interface for messages
 * @description
 * Returns a `Logger` that has `.log`, `.error`, `.warn` and `.debug` (controlled with environment `DEBUG=typeA,typeB`) methods. Logging is done with a consistent prefix (type of logger, date) followed by the actual message using the underlying console.
 * @example
 * <BR>
 *
 * ```javascript
 * import { logger } from '@polkadot';
 *
 * const l = logger('test');
 * ```
 */
export declare function logger(_type: string): Logger;
