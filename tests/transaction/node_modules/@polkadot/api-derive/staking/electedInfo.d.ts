import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveStakingElected, StakingQueryFlags } from '../types';
export declare function electedInfo(instanceId: string, api: DeriveApi): (flags?: StakingQueryFlags) => Observable<DeriveStakingElected>;
