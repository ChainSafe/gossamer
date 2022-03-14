import type { Observable } from 'rxjs';
import type { PalletStakingEraRewardPoints } from '@polkadot/types/lookup';
import type { DeriveApi } from '../types';
/**
 * @description Retrieve the staking overview, including elected and points earned
 */
export declare function currentPoints(instanceId: string, api: DeriveApi): () => Observable<PalletStakingEraRewardPoints>;
