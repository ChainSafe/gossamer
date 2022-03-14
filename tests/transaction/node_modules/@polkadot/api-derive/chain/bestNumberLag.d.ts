import type { Observable } from 'rxjs';
import type { BlockNumber } from '@polkadot/types/interfaces';
import type { DeriveApi } from '../types';
/**
 * @name bestNumberLag
 * @returns A number of blocks
 * @description Calculates the lag between finalized head and best head
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.chain.bestNumberLag((lag) => {
 *   console.log(`finalized is ${lag} blocks behind head`);
 * });
 * ```
 */
export declare function bestNumberLag(instanceId: string, api: DeriveApi): () => Observable<BlockNumber>;
