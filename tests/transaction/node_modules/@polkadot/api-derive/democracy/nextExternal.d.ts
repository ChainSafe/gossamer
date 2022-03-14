import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveProposalExternal } from '../types';
export declare function nextExternal(instanceId: string, api: DeriveApi): () => Observable<DeriveProposalExternal | null>;
