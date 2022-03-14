import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveStakingWaiting, StakingQueryFlags } from '../types';
export declare function waitingInfo(instanceId: string, api: DeriveApi): (flags?: StakingQueryFlags) => Observable<DeriveStakingWaiting>;
