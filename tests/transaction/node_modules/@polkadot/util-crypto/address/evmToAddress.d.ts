import type { HexString } from '@polkadot/util/types';
import type { HashType } from '../secp256k1/types';
import type { Prefix } from './types';
/**
 * @name evmToAddress
 * @summary Converts an EVM address to its corresponding SS58 address.
 */
export declare function evmToAddress(evmAddress: HexString | string | Uint8Array, ss58Format?: Prefix, hashType?: HashType): string;
