declare type LazySection<T> = Record<string, T>;
declare type LazyRecord<T> = Record<string, LazySection<T>>;
export declare function lazyDeriveSection<T>(result: LazyRecord<T>, section: string, getKeys: (s: string) => string[], creator: (s: string, m: string) => T): void;
export {};
