import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveStakingValidators } from '../types';
export declare function nextElected(instanceId: string, api: DeriveApi): () => Observable<AccountId[]>;
/**
 * @description Retrieve latest list of validators
 */
export declare function validators(instanceId: string, api: DeriveApi): () => Observable<DeriveStakingValidators>;
