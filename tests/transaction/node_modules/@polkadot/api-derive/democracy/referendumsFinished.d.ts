import type { Observable } from 'rxjs';
import type { PalletDemocracyReferendumInfo } from '@polkadot/types/lookup';
import type { DeriveApi } from '../types';
declare type ReferendumInfoFinished = PalletDemocracyReferendumInfo['asFinished'];
export declare function referendumsFinished(instanceId: string, api: DeriveApi): () => Observable<ReferendumInfoFinished[]>;
export {};
