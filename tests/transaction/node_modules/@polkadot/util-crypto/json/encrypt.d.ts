import type { EncryptedJson } from './types';
export declare function jsonEncrypt(data: Uint8Array, contentType: string[], passphrase?: string | null): EncryptedJson;
