import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
export declare function isAddress(address?: HexString | string | null, ignoreChecksum?: boolean, ss58Format?: Prefix): address is string;
