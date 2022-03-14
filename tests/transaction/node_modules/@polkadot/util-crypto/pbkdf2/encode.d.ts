/// <reference types="node" />
import type { HexString } from '@polkadot/util/types';
interface Result {
    password: Uint8Array;
    rounds: number;
    salt: Uint8Array;
}
export declare function pbkdf2Encode(passphrase?: HexString | Buffer | Uint8Array | string, salt?: Buffer | Uint8Array, rounds?: number, onlyJs?: boolean): Result;
export {};
