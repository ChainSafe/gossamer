import type { HexString } from '@polkadot/util/types';
export declare function sr25519DerivePublic(publicKey: HexString | Uint8Array | string, chainCode: Uint8Array): Uint8Array;
