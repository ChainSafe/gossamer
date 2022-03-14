// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { switchMap } from 'rxjs';
import { memo } from "../util/index.js";
export function getBlockByNumber(instanceId, api) {
  return memo(instanceId, blockNumber => api.rpc.chain.getBlockHash(blockNumber).pipe(switchMap(h => api.derive.chain.getBlock(h))));
}