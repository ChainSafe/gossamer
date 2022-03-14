import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
/**
 * @name checkAddress
 * @summary Validates an ss58 address.
 * @description
 * From the provided input, validate that the address is a valid input.
 */
export declare function checkAddress(address: HexString | string, prefix: Prefix): [boolean, string | null];
