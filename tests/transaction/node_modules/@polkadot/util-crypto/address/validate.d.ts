import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
export declare function validateAddress(encoded?: HexString | string | null, ignoreChecksum?: boolean, ss58Format?: Prefix): encoded is string;
