import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveParachain } from '../types';
export declare function overview(instanceId: string, api: DeriveApi): () => Observable<DeriveParachain[]>;
