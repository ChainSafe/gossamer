// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, switchMap } from 'rxjs';
import { memo } from "../util/index.js";
export function events(instanceId, api) {
  return memo(instanceId, blockHash => combineLatest([api.rpc.chain.getBlock(blockHash), api.queryAt(blockHash).pipe(switchMap(queryAt => queryAt.system.events()))]).pipe(map(([block, events]) => ({
    block,
    events
  }))));
}