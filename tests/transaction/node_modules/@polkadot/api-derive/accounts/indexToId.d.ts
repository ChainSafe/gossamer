import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
/**
 * @name indexToId
 * @param {( AccountIndex | string )} accountIndex - An accounts index in different formats.
 * @returns Returns the corresponding AccountId.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.accounts.indexToId('F7Hs', (accountId) => {
 *   console.log(`The AccountId of F7Hs is ${accountId}`);
 * });
 * ```
 */
export declare function indexToId(instanceId: string, api: DeriveApi): (accountIndex: AccountIndex | string) => Observable<AccountId | undefined>;
