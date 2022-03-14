import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveSocietyMember } from '../types';
export declare function _members(instanceId: string, api: DeriveApi): (accountIds: AccountId[]) => Observable<DeriveSocietyMember[]>;
/**
 * @description Get the member info for a society
 */
export declare function members(instanceId: string, api: DeriveApi): () => Observable<DeriveSocietyMember[]>;
