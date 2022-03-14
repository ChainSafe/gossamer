import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveDispatch } from '../types';
export declare function dispatchQueue(instanceId: string, api: DeriveApi): () => Observable<DeriveDispatch[]>;
