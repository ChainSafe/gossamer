import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
import type { Collective } from './types';
export type { AccountId } from '@polkadot/types/interfaces';
export declare function prime(section: Collective): (instanceId: string, api: DeriveApi) => () => Observable<AccountId | null>;
