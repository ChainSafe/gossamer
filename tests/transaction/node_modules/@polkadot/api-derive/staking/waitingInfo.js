// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, switchMap } from 'rxjs';
import { memo } from "../util/index.js";
const DEFAULT_FLAGS = {
  withController: true,
  withPrefs: true
};
export function waitingInfo(instanceId, api) {
  return memo(instanceId, (flags = DEFAULT_FLAGS) => combineLatest([api.derive.staking.validators(), api.derive.staking.stashes()]).pipe(switchMap(([{
    nextElected
  }, stashes]) => {
    const elected = nextElected.map(a => a.toString());
    const waiting = stashes.filter(v => !elected.includes(v.toString()));
    return api.derive.staking.queryMulti(waiting, flags).pipe(map(info => ({
      info,
      waiting
    })));
  })));
}