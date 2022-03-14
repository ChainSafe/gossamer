import type { HexString } from '@polkadot/util/types';
import type { Keypair } from '../types';
/**
 * @name ed25519Sign
 * @summary Signs a message using the supplied secretKey
 * @description
 * Returns message signature of `message`, using the `secretKey`.
 * @example
 * <BR>
 *
 * ```javascript
 * import { ed25519Sign } from '@polkadot/util-crypto';
 *
 * ed25519Sign([...], [...]); // => [...]
 * ```
 */
export declare function ed25519Sign(message: HexString | Uint8Array | string, { publicKey, secretKey }: Partial<Keypair>, onlyJs?: boolean): Uint8Array;
