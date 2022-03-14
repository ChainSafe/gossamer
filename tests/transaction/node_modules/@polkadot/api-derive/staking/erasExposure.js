// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js";
import { getEraCache, setEraCache } from "./cache.js";
import { combineEras, erasHistoricApply, singleEra } from "./util.js";
const CACHE_KEY = 'eraExposure';

function mapStakers(era, stakers) {
  const nominators = {};
  const validators = {};
  stakers.forEach(([key, exposure]) => {
    const validatorId = key.args[1].toString();
    validators[validatorId] = exposure;
    exposure.others.forEach(({
      who
    }, validatorIndex) => {
      const nominatorId = who.toString();
      nominators[nominatorId] = nominators[nominatorId] || [];
      nominators[nominatorId].push({
        validatorId,
        validatorIndex
      });
    });
  });
  return {
    era,
    nominators,
    validators
  };
}

export function _eraExposure(instanceId, api) {
  return memo(instanceId, (era, withActive = false) => {
    const [cacheKey, cached] = getEraCache(CACHE_KEY, era, withActive);
    return cached ? of(cached) : api.query.staking.erasStakersClipped.entries(era).pipe(map(r => setEraCache(cacheKey, withActive, mapStakers(era, r))));
  });
}
export const eraExposure = singleEra('_eraExposure');
export const _erasExposure = combineEras('_eraExposure');
export const erasExposure = erasHistoricApply('_erasExposure');