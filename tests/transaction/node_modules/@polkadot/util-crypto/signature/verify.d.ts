import type { HexString } from '@polkadot/util/types';
import type { VerifyResult } from '../types';
export declare function signatureVerify(message: HexString | Uint8Array | string, signature: HexString | Uint8Array | string, addressOrPublicKey: HexString | Uint8Array | string): VerifyResult;
