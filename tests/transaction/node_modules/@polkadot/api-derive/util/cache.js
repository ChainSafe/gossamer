// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { deriveNoopCache } from "./cacheImpl.js";
const CHACHE_EXPIRY = 7 * (24 * 60) * (60 * 1000);
let deriveCache;

function wrapCache(keyStart, cache) {
  return {
    del: partial => cache.del(`${keyStart}${partial}`),
    forEach: cache.forEach,
    get: partial => {
      const key = `${keyStart}${partial}`;
      const cached = cache.get(key);

      if (cached) {
        cached.x = Date.now();
        cache.set(key, cached);
        return cached.v;
      }

      return undefined;
    },
    set: (partial, v) => {
      cache.set(`${keyStart}${partial}`, {
        v,
        x: Date.now()
      });
    }
  };
}

function clearCache(cache) {
  // clear all expired values
  const now = Date.now();
  const all = [];
  cache.forEach((key, {
    x
  }) => {
    now - x > CHACHE_EXPIRY && all.push(key);
  }); // don't do delete inside loop, just in-case

  all.forEach(key => cache.del(key));
}

export function setDeriveCache(prefix = '', cache) {
  deriveCache = cache ? wrapCache(`derive:${prefix}:`, cache) : deriveNoopCache;

  if (cache) {
    clearCache(cache);
  }
}
setDeriveCache();
export { deriveCache };