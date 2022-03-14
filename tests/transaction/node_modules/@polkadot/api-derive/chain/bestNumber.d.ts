import type { DeriveApi } from '../types';
/**
 * @name bestNumber
 * @returns The latest block number.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.chain.bestNumber((blockNumber) => {
 *   console.log(`the current best block is #${blockNumber}`);
 * });
 * ```
 */
export declare const bestNumber: (instanceId: string, api: DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types/interfaces/runtime/types").BlockNumber>;
