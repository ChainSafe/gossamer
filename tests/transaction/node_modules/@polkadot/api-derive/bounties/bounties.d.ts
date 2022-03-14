import type { Observable } from 'rxjs';
import type { DeriveApi, DeriveBounties } from '../types';
export declare function bounties(instanceId: string, api: DeriveApi): () => Observable<DeriveBounties>;
