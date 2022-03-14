// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, of } from 'rxjs';
import { memo } from "../util/index.js";
import { getEraCache, setEraCache } from "./cache.js";
import { combineEras, erasHistoricApply, singleEra } from "./util.js";
const CACHE_KEY = 'eraSlashes';

function mapSlashes(era, noms, vals) {
  const nominators = {};
  const validators = {};
  noms.forEach(([key, optBalance]) => {
    nominators[key.args[1].toString()] = optBalance.unwrap();
  });
  vals.forEach(([key, optRes]) => {
    validators[key.args[1].toString()] = optRes.unwrapOrDefault()[1];
  });
  return {
    era,
    nominators,
    validators
  };
}

export function _eraSlashes(instanceId, api) {
  return memo(instanceId, (era, withActive) => {
    const [cacheKey, cached] = getEraCache(CACHE_KEY, era, withActive);
    return cached ? of(cached) : combineLatest([api.query.staking.nominatorSlashInEra.entries(era), api.query.staking.validatorSlashInEra.entries(era)]).pipe(map(([n, v]) => setEraCache(cacheKey, withActive, mapSlashes(era, n, v))));
  });
}
export const eraSlashes = singleEra('_eraSlashes');
export const _erasSlashes = combineEras('_eraSlashes');
export const erasSlashes = erasHistoricApply('_erasSlashes');