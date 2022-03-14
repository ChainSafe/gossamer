// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { BN_ZERO } from '@polkadot/util';
import { memo } from "../util/index.js";
import { filterCachedEras, getEraMultiCache, setEraMultiCache } from "./cache.js";
import { erasHistoricApply, filterEras } from "./util.js";
const CACHE_KEY = 'eraPoints';

function mapValidators({
  individual
}) {
  return [...individual.entries()].filter(([, points]) => points.gt(BN_ZERO)).reduce((result, [validatorId, points]) => {
    result[validatorId.toString()] = points;
    return result;
  }, {});
}

function mapPoints(eras, points) {
  return eras.map((era, index) => ({
    era,
    eraPoints: points[index].total,
    validators: mapValidators(points[index])
  }));
}

export function _erasPoints(instanceId, api) {
  return memo(instanceId, (eras, withActive) => {
    if (!eras.length) {
      return of([]);
    }

    const cached = getEraMultiCache(CACHE_KEY, eras, withActive);
    const remaining = filterEras(eras, cached);
    return !remaining.length ? of(cached) : api.query.staking.erasRewardPoints.multi(remaining).pipe(map(p => filterCachedEras(eras, cached, setEraMultiCache(CACHE_KEY, withActive, mapPoints(remaining, p)))));
  });
}
export const erasPoints = erasHistoricApply('_erasPoints');