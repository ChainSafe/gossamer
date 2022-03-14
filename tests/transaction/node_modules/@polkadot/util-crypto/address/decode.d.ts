import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
export declare function decodeAddress(encoded?: HexString | string | Uint8Array | null, ignoreChecksum?: boolean, ss58Format?: Prefix): Uint8Array;
