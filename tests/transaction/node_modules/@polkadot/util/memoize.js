// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isUndefined } from "./is/undefined.js";
import { stringify } from "./stringify.js";

function defaultGetId() {
  return 'none';
} // eslint-disable-next-line @typescript-eslint/no-explicit-any


export function memoize(fn, {
  getInstanceId = defaultGetId
} = {}) {
  const cache = {};

  const memoized = (...args) => {
    const stringParams = stringify(args);
    const instanceId = getInstanceId();

    if (!cache[instanceId]) {
      cache[instanceId] = {};
    }

    if (isUndefined(cache[instanceId][stringParams])) {
      cache[instanceId][stringParams] = fn(...args);
    }

    return cache[instanceId][stringParams];
  };

  memoized.unmemoize = (...args) => {
    const stringParams = stringify(args);
    const instanceId = getInstanceId();

    if (cache[instanceId] && !isUndefined(cache[instanceId][stringParams])) {
      delete cache[instanceId][stringParams];
    }
  };

  return memoized;
}