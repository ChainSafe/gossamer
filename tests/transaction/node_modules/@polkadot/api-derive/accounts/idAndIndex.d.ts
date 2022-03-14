import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex, Address } from '@polkadot/types/interfaces';
import type { AccountIdAndIndex, DeriveApi } from '../types';
/**
 * @name idAndIndex
 * @param {(Address | AccountId | AccountIndex | Uint8Array | string | null)} address - An accounts address in various formats.
 * @description  An array containing the [[AccountId]] and [[AccountIndex]] as optional values.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.accounts.idAndIndex('F7Hs', ([id, ix]) => {
 *   console.log(`AccountId #${id} with corresponding AccountIndex ${ix}`);
 * });
 * ```
 */
export declare function idAndIndex(instanceId: string, api: DeriveApi): (address?: Address | AccountId | AccountIndex | Uint8Array | string | null) => Observable<AccountIdAndIndex>;
