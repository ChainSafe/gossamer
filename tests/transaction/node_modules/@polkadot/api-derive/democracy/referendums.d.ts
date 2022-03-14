import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveReferendumExt } from '../types';
export declare function referendums(instanceId: string, api: DeriveApi): () => Observable<DeriveReferendumExt[]>;
