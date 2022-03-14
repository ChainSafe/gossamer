// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, of, switchMap } from 'rxjs';
import { memo } from "../util/index.js";

/**
 * @description Get the candidate info for a society
 */
export function candidates(instanceId, api) {
  return memo(instanceId, () => api.query.society.candidates().pipe(switchMap(candidates => combineLatest([of(candidates), api.query.society.suspendedCandidates.multi(candidates.map(({
    who
  }) => who))])), map(([candidates, suspended]) => candidates.map(({
    kind,
    value,
    who
  }, index) => ({
    accountId: who,
    isSuspended: suspended[index].isSome,
    kind,
    value
  })))));
}