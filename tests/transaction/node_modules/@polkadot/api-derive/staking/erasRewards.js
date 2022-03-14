// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js";
import { filterCachedEras, getEraMultiCache, setEraMultiCache } from "./cache.js";
import { erasHistoricApply, filterEras } from "./util.js";
const CACHE_KEY = 'eraRewards';

function mapRewards(eras, optRewards) {
  return eras.map((era, index) => ({
    era,
    eraReward: optRewards[index].unwrapOrDefault()
  }));
}

export function _erasRewards(instanceId, api) {
  return memo(instanceId, (eras, withActive) => {
    if (!eras.length) {
      return of([]);
    }

    const cached = getEraMultiCache(CACHE_KEY, eras, withActive);
    const remaining = filterEras(eras, cached);

    if (!remaining.length) {
      return of(cached);
    }

    return api.query.staking.erasValidatorReward.multi(remaining).pipe(map(r => filterCachedEras(eras, cached, setEraMultiCache(CACHE_KEY, withActive, mapRewards(remaining, r)))));
  });
}
export const erasRewards = erasHistoricApply('_erasRewards');