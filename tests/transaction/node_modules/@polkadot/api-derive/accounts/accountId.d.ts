import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex, Address } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
/**
 * @name accountId
 * @param {(Address | AccountId | AccountIndex | string | null)} address - An accounts address in various formats.
 * @description  An [[AccountId]]
 */
export declare function accountId(instanceId: string, api: DeriveApi): (address?: Address | AccountId | AccountIndex | string | null) => Observable<AccountId>;
