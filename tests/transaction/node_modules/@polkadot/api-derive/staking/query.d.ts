import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveStakingQuery, StakingQueryFlags } from '../types';
/**
 * @description From a stash, retrieve the controllerId and all relevant details
 */
export declare const query: (instanceId: string, api: DeriveApi) => (accountId: string | Uint8Array, flags: StakingQueryFlags) => Observable<DeriveStakingQuery>;
export declare function queryMulti(instanceId: string, api: DeriveApi): (accountIds: (Uint8Array | string)[], flags: StakingQueryFlags) => Observable<DeriveStakingQuery[]>;
