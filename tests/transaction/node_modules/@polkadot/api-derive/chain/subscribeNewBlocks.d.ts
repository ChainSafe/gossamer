import type { Observable } from 'rxjs';
import type { SignedBlockExtended } from '../type/types';
import type { DeriveApi } from '../types';
/**
 * @name subscribeNewBlocks
 * @returns The latest block & events for that block
 */
export declare function subscribeNewBlocks(instanceId: string, api: DeriveApi): () => Observable<SignedBlockExtended>;
