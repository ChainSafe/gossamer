import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveDemocracyLock } from '../types';
export declare function locks(instanceId: string, api: DeriveApi): (accountId: string | AccountId) => Observable<DeriveDemocracyLock[]>;
