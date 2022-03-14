import type { Constructor } from '../types';
/**
 * @name isChildClass
 * @summary Tests if the child extends the parent Class
 * @description
 * Checks to see if the child Class extends the parent Class
 * @example
 * <BR>
 *
 * ```javascript
 * import { isChildClass } from '@polkadot/util';
 *
 * console.log('isChildClass', isChildClass(BN, BN); // => true
 * console.log('isChildClass', isChildClass(BN, Uint8Array); // => false
 * ```
 */
export declare function isChildClass<P extends Constructor>(Parent: P, Child?: Constructor | null | unknown): Child is P;
