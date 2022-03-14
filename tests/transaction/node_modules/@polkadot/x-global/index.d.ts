export { packageInfo } from './packageInfo';
declare type GlobalNames = keyof typeof window;
declare type GlobalType<T extends keyof typeof window> = typeof window[T];
export declare const xglobal: typeof globalThis;
export declare function extractGlobal<N extends GlobalNames, T extends GlobalType<N>>(name: N, fallback: unknown): T;
export declare function exposeGlobal<N extends GlobalNames>(name: N, fallback: unknown): void;
