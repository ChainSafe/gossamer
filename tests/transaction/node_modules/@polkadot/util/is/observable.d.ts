import type { Observable } from '../types';
/**
 * @name isBObservable
 * @summary Tests for a `Observable` object instance.
 * @description
 * Checks to see if the input object is an instance of `BN` (bn.js).
 * @example
 * <BR>
 *
 * ```javascript
 * import { isObservable } from '@polkadot/util';
 *
 * console.log('isObservable', isObservable(...));
 * ```
 */
export declare const isObservable: (value?: unknown) => value is Observable;
