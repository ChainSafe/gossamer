import type { HexString } from '@polkadot/util/types';
import type { Prefix } from './types';
/**
 * @name deriveAddress
 * @summary Creates a sr25519 derived address from the supplied and path.
 * @description
 * Creates a sr25519 derived address based on the input address/publicKey and the uri supplied.
 */
export declare function deriveAddress(who: HexString | Uint8Array | string, suri: string, ss58Format?: Prefix): string;
