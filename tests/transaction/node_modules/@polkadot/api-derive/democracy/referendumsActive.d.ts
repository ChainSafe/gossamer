import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveReferendum } from '../types';
export declare function referendumsActive(instanceId: string, api: DeriveApi): () => Observable<DeriveReferendum[]>;
