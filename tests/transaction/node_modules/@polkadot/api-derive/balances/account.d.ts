import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex, Address } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveBalancesAccount } from '../types';
/**
 * @name account
 * @param {( AccountIndex | AccountId | Address | string )} address - An accounts Id in different formats.
 * @returns An object containing the results of various balance queries
 * @example
 * <BR>
 *
 * ```javascript
 * const ALICE = 'F7Hs';
 *
 * api.derive.balances.all(ALICE, ({ accountId, lockedBalance }) => {
 *   console.log(`The account ${accountId} has a locked balance ${lockedBalance} units.`);
 * });
 * ```
 */
export declare function account(instanceId: string, api: DeriveApi): (address: AccountIndex | AccountId | Address | string) => Observable<DeriveBalancesAccount>;
