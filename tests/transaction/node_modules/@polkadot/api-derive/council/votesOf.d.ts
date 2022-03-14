import type { Observable } from 'rxjs';
import type { AccountId } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveCouncilVote } from '../types';
export declare function votesOf(instanceId: string, api: DeriveApi): (accountId: string | Uint8Array | AccountId) => Observable<DeriveCouncilVote>;
