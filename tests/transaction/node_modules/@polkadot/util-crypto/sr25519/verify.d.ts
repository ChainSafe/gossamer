import type { HexString } from '@polkadot/util/types';
/**
 * @name sr25519Verify
 * @description Verifies the signature of `message`, using the supplied pair
 */
export declare function sr25519Verify(message: HexString | Uint8Array | string, signature: HexString | Uint8Array | string, publicKey: HexString | Uint8Array | string): boolean;
