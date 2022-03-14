import type { HexString } from '@polkadot/util/types';
/**
 * @name addressToEvm
 * @summary Converts an SS58 address to its corresponding EVM address.
 */
export declare function addressToEvm(address: HexString | string | Uint8Array, ignoreChecksum?: boolean): Uint8Array;
