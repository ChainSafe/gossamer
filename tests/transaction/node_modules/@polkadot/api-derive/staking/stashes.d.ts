import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
/**
 * @description Retrieve the list of all validator stashes
 */
export declare function stashes(instanceId: string, api: DeriveApi): () => Observable<AccountId[]>;
