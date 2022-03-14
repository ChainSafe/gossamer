import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveTreasuryProposals } from '../types';
/**
 * @description Retrieve all active and approved treasury proposals, along with their info
 */
export declare function proposals(instanceId: string, api: DeriveApi): () => Observable<DeriveTreasuryProposals>;
