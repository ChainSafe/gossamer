import type { AnyString } from '../types';
/**
 * @name stringLowerFirst
 * @summary Lowercase the first letter of a string
 * @description
 * Lowercase the first letter of a string
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringLowerFirst } from '@polkadot/util';
 *
 * stringLowerFirst('ABC'); // => 'aBC'
 * ```
 */
export declare const stringLowerFirst: (value?: AnyString | null | undefined) => string;
/**
 * @name stringUpperFirst
 * @summary Uppercase the first letter of a string
 * @description
 * Lowercase the first letter of a string
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringUpperFirst } from '@polkadot/util';
 *
 * stringUpperFirst('abc'); // => 'Abc'
 * ```
 */
export declare const stringUpperFirst: (value?: AnyString | null | undefined) => string;
