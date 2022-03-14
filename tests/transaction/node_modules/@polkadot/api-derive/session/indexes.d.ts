import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveSessionIndexes } from '../types';
export declare function indexes(instanceId: string, api: DeriveApi): () => Observable<DeriveSessionIndexes>;
