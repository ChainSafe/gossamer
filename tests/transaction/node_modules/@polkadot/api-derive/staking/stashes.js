// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, startWith, switchMap } from 'rxjs';
import { drr, memo } from "../util/index.js";

function onBondedEvent(api) {
  let current = Date.now();
  return api.query.system.events().pipe(map(events => {
    current = events.filter(({
      event,
      phase
    }) => {
      try {
        return phase.isApplyExtrinsic && event.section === 'staking' && event.method === 'Bonded';
      } catch {
        return false;
      }
    }) ? Date.now() : current;
    return current;
  }), startWith(current), drr({
    skipTimeout: true
  }));
}
/**
 * @description Retrieve the list of all validator stashes
 */


export function stashes(instanceId, api) {
  return memo(instanceId, () => onBondedEvent(api).pipe(switchMap(() => api.query.staking.validators.keys()), map(keys => keys.map(({
    args: [v]
  }) => v).filter(a => a))));
}