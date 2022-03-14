// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map } from 'rxjs';
import { memo } from "../util/index.js";
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

export function bestNumberLag(instanceId, api) {
  return memo(instanceId, () => combineLatest([api.derive.chain.bestNumber(), api.derive.chain.bestNumberFinalized()]).pipe(map(([bestNumber, bestNumberFinalized]) => api.registry.createType('BlockNumber', bestNumber.sub(bestNumberFinalized)))));
}