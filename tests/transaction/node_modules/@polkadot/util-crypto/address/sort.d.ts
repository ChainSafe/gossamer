import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
export declare function sortAddresses(addresses: (HexString | Uint8Array | string)[], ss58Format?: Prefix): string[];
