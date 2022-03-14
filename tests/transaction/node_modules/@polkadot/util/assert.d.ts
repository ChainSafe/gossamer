declare type MessageFn = () => string;
/**
 * @name assert
 * @summary Checks for a valid test, if not Error is thrown.
 * @description
 * Checks that `test` is a truthy value. If value is falsy (`null`, `undefined`, `false`, ...), it throws an Error with the supplied `message`. When `test` passes, `true` is returned.
 * @example
 * <BR>
 *
 * ```javascript
 * const { assert } from '@polkadot/util';
 *
 * assert(true, 'True should be true'); // passes
 * assert(false, 'False should not be true'); // Error thrown
 * assert(false, () => 'message'); // Error with 'message'
 * ```
 */
export declare function assert(condition: unknown, message: string | MessageFn): asserts condition;
/**
 * @name assertReturn
 * @description Returns when the value is not undefined/null, otherwise throws assertion error
 */
export declare function assertReturn<T>(value: T | undefined | null, message: string | MessageFn): T;
/**
 * @name assertUnreachable
 * @description An assertion helper that ensures all codepaths are followed
 */
export declare function assertUnreachable(x: never): never;
export {};
