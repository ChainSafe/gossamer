import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveProposal } from '../types';
export declare function proposals(instanceId: string, api: DeriveApi): () => Observable<DeriveProposal[]>;
