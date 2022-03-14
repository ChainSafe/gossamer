import type { Observable } from 'rxjs';
import type { AccountId, AccountIndex, Address } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveBalancesAccount } from '../types';
export declare function votingBalances(instanceId: string, api: DeriveApi): (addresses?: (AccountId | AccountIndex | Address | string)[]) => Observable<DeriveBalancesAccount[]>;
