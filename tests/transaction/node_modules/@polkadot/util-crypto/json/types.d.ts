import type { HexString } from '@polkadot/util/types';
export declare type EncryptedJsonVersion = '0' | '1' | '2' | '3';
export declare type EncryptedJsonEncoding = 'none' | 'scrypt' | 'xsalsa20-poly1305';
export interface EncryptedJsonDescriptor {
    content: string[];
    type: EncryptedJsonEncoding | EncryptedJsonEncoding[];
    version: EncryptedJsonVersion;
}
export interface EncryptedJson {
    encoded: HexString | string;
    encoding: EncryptedJsonDescriptor;
}
