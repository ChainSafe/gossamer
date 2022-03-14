import type { HexString } from '@polkadot/util/types';
import type { Keypair } from '../types';
/**
 * @name sr25519VrfSign
 * @description Sign with sr25519 vrf signing (deterministic)
 */
export declare function sr25519VrfSign(message: HexString | Uint8Array | string, { secretKey }: Partial<Keypair>, context?: HexString | string | Uint8Array, extra?: HexString | string | Uint8Array): Uint8Array;
