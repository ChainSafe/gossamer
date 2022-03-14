/// <reference types="bn.js" />
import type { BN } from '@polkadot/util';
import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
/**
 * @name encodeMultiAddress
 * @summary Creates a multisig address.
 * @description
 * Creates a Substrate multisig address based on the input address and the required threshold.
 */
export declare function encodeMultiAddress(who: (HexString | Uint8Array | string)[], threshold: bigint | BN | number, ss58Format?: Prefix): string;
