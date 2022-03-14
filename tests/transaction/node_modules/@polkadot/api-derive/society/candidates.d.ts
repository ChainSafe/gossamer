import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveSocietyCandidate } from '../types';
/**
 * @description Get the candidate info for a society
 */
export declare function candidates(instanceId: string, api: DeriveApi): () => Observable<DeriveSocietyCandidate[]>;
