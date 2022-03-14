import type { Observable } from 'rxjs';
import type { DeriveApi } from '../types';
import type { DeriveStakingKeys } from './types';
export declare const keys: (instanceId: string, api: DeriveApi) => (stashId: string | Uint8Array) => Observable<DeriveStakingKeys>;
export declare function keysMulti(instanceId: string, api: DeriveApi): (stashIds: (Uint8Array | string)[]) => Observable<DeriveStakingKeys[]>;
