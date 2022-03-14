import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveSociety } from '../types';
/**
 * @description Get the overall info for a society
 */
export declare function info(instanceId: string, api: DeriveApi): () => Observable<DeriveSociety>;
