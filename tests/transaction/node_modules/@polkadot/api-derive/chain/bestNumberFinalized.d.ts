import type { DeriveApi } from '../types';
/**
 * @name bestNumberFinalized
 * @returns A BlockNumber
 * @description Get the latest finalized block number.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.chain.bestNumberFinalized((blockNumber) => {
 *   console.log(`the current finalized block is #${blockNumber}`);
 * });
 * ```
 */
export declare const bestNumberFinalized: (instanceId: string, api: DeriveApi) => () => import("rxjs").Observable<import("@polkadot/types/interfaces/runtime/types").BlockNumber>;
