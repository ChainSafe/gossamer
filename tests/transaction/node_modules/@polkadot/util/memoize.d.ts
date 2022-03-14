import type { Memoized } from './types';
interface Options {
    getInstanceId?: () => string;
}
export declare function memoize<T, F extends (...args: any[]) => T>(fn: F, { getInstanceId }?: Options): Memoized<F>;
export {};
