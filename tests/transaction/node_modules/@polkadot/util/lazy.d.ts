declare type AnyFn = (...args: unknown[]) => unknown;
export declare function lazyMethod<T, K>(result: Record<string, T> | AnyFn, item: K, creator: (d: K) => T, getName?: (d: K) => string): void;
export declare function lazyMethods<T, K>(result: Record<string, T>, items: K[], creator: (v: K) => T, getName?: (m: K) => string): Record<string, T>;
export {};
