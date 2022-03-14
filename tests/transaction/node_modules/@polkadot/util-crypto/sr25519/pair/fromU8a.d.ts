import type { HexString } from '@polkadot/util/types';
import type { Keypair } from '../../types';
export declare function sr25519PairFromU8a(full: HexString | Uint8Array | string): Keypair;
