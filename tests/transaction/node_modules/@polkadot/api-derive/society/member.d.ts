import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveSocietyMember } from '../types';
/**
 * @description Get the member info for a society
 */
export declare function member(instanceId: string, api: DeriveApi): (accountId: AccountId) => Observable<DeriveSocietyMember>;
