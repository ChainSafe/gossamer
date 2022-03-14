import type { Observable } from 'rxjs';
import type { BlockNumber } from '@polkadot/types/interfaces';
import type { DeriveApi, DeriveSessionProgress } from '../types';
/**
 * @description Retrieves all the session and era query and calculates specific values on it as the length of the session and eras
 */
export declare function progress(instanceId: string, api: DeriveApi): () => Observable<DeriveSessionProgress>;
export declare const eraLength: (instanceId: string, api: DeriveApi) => () => Observable<BlockNumber>;
export declare const eraProgress: (instanceId: string, api: DeriveApi) => () => Observable<BlockNumber>;
export declare const sessionProgress: (instanceId: string, api: DeriveApi) => () => Observable<BlockNumber>;
