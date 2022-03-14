// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js";
import { getEraCache, setEraCache } from "./cache.js";
import { combineEras, erasHistoricApply, singleEra } from "./util.js";
const CACHE_KEY = 'eraPrefs';

function mapPrefs(era, all) {
  const validators = {};
  all.forEach(([key, prefs]) => {
    validators[key.args[1].toString()] = prefs;
  });
  return {
    era,
    validators
  };
}

export function _eraPrefs(instanceId, api) {
  return memo(instanceId, (era, withActive) => {
    const [cacheKey, cached] = getEraCache(CACHE_KEY, era, withActive);
    return cached ? of(cached) : api.query.staking.erasValidatorPrefs.entries(era).pipe(map(r => setEraCache(cacheKey, withActive, mapPrefs(era, r))));
  });
}
export const eraPrefs = singleEra('_eraPrefs');
export const _erasPrefs = combineEras('_eraPrefs');
export const erasPrefs = erasHistoricApply('_erasPrefs');