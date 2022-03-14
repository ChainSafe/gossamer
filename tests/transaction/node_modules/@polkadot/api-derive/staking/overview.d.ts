import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveStakingOverview } from '../types';
/**
 * @description Retrieve the staking overview, including elected and points earned
 */
export declare function overview(instanceId: string, api: DeriveApi): () => Observable<DeriveStakingOverview>;
