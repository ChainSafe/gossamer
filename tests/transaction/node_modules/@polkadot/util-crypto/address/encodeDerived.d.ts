/// <reference types="bn.js" />
import type { BN } from '@polkadot/util';
import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
/**
 * @name encodeDerivedAddress
 * @summary Creates a derived address as used in Substrate utility.
 * @description
 * Creates a Substrate derived address based on the input address/publicKey and the index supplied.
 */
export declare function encodeDerivedAddress(who: HexString | Uint8Array | string, index: bigint | BN | number, ss58Format?: Prefix): string;
