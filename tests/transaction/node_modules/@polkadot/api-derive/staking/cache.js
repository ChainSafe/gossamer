// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { deriveCache } from "../util/index.js";
export function getEraCache(CACHE_KEY, era, withActive) {
  const cacheKey = `${CACHE_KEY}-${era.toString()}`;
  return [cacheKey, withActive ? undefined : deriveCache.get(cacheKey)];
}
export function getEraMultiCache(CACHE_KEY, eras, withActive) {
  const cached = withActive ? [] : eras.map(e => deriveCache.get(`${CACHE_KEY}-${e.toString()}`)).filter(v => !!v);
  return cached;
}
export function setEraCache(cacheKey, withActive, value) {
  !withActive && deriveCache.set(cacheKey, value);
  return value;
}
export function setEraMultiCache(CACHE_KEY, withActive, values) {
  !withActive && values.forEach(v => deriveCache.set(`${CACHE_KEY}-${v.era.toString()}`, v));
  return values;
}
export function filterCachedEras(eras, cached, query) {
  return eras.map(e => cached.find(({
    era
  }) => e.eq(era)) || query.find(({
    era
  }) => e.eq(era)));
}