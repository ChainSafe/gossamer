// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js"; // parse into Indexes

function parse([currentIndex, activeEra, activeEraStart, currentEra, validatorCount]) {
  return {
    activeEra,
    activeEraStart,
    currentEra,
    currentIndex,
    validatorCount
  };
} // query based on latest


function queryStaking(api) {
  return api.queryMulti([api.query.session.currentIndex, api.query.staking.activeEra, api.query.staking.currentEra, api.query.staking.validatorCount]).pipe(map(([currentIndex, activeOpt, currentEra, validatorCount]) => {
    const {
      index,
      start
    } = activeOpt.unwrapOrDefault();
    return parse([currentIndex, index, start, currentEra.unwrapOrDefault(), validatorCount]);
  }));
} // query based on latest


function querySession(api) {
  return api.query.session.currentIndex().pipe(map(currentIndex => parse([currentIndex, api.registry.createType('EraIndex'), api.registry.createType('Option<Moment>'), api.registry.createType('EraIndex'), api.registry.createType('u32')])));
} // empty set when none is available


function empty(api) {
  return of(parse([api.registry.createType('SessionIndex', 1), api.registry.createType('EraIndex'), api.registry.createType('Option<Moment>'), api.registry.createType('EraIndex'), api.registry.createType('u32')]));
}

export function indexes(instanceId, api) {
  return memo(instanceId, () => api.query.session ? api.query.staking ? queryStaking(api) : querySession(api) : empty(api));
}