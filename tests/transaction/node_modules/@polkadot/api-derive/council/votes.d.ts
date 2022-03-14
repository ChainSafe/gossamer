import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveCouncilVotes } from '../types';
export declare function votes(instanceId: string, api: DeriveApi): () => Observable<DeriveCouncilVotes>;
