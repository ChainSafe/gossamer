import type { EncryptedJson } from './types';
export declare function jsonDecrypt({ encoded, encoding }: EncryptedJson, passphrase?: string | null): Uint8Array;
