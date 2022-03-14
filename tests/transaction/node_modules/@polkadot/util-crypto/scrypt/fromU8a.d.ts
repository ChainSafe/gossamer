import type { Params } from './types';
interface Result {
    params: Params;
    salt: Uint8Array;
}
export declare function scryptFromU8a(data: Uint8Array): Result;
export {};
