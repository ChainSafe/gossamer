import type { HexString } from '@polkadot/util/types';
import type { Params } from './types';
interface Result {
    params: Params;
    password: Uint8Array;
    salt: Uint8Array;
}
export declare function scryptEncode(passphrase?: HexString | Uint8Array | string, salt?: Uint8Array, params?: {
    N: number;
    p: number;
    r: number;
}, onlyJs?: boolean): Result;
export {};
