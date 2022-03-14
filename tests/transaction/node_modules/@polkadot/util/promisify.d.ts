/**
 * @name promisify
 * @summary Wraps an async callback into a `Promise`
 * @description
 * Wraps the supplied async function `fn` that has a standard JS callback `(error: Error, result: any)` into a `Promise`, passing the supplied parameters. When `error` is set, the Promise is rejected, else the Promise resolves with the `result` value.
 * @example
 * <BR>
 *
 * ```javascript
 * const { promisify } from '@polkadot/util';
 *
 * await promisify(null, ((a, cb) => cb(null, a), true); // resolves with `true`
 * await promisify(null, (cb) => cb(new Error('error!'))); // rejects with `error!`
 * ```
 */
export declare function promisify<R = any>(self: unknown, fn: (...params: any) => any, ...params: any[]): Promise<R>;
