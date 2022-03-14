import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
/**
 * @name idToIndex
 * @param {( AccountId | string )} accountId - An accounts Id in different formats.
 * @returns Returns the corresponding AccountIndex.
 * @example
 * <BR>
 *
 * ```javascript
 * const ALICE = '5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY';
 * api.derive.accounts.idToIndex(ALICE, (accountIndex) => {
 *   console.log(`The AccountIndex of ${ALICE} is ${accountIndex}`);
 * });
 * ```
 */
export declare function idToIndex(instanceId: string, api: DeriveApi): (accountId: AccountId | string) => Observable<AccountIndex | undefined>;
