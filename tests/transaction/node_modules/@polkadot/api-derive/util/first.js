// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from '@polkadot/rpc-core';
export function firstObservable(obs) {
  return obs.pipe(map(([a]) => a));
}
export function firstMemo(fn) {
  return (instanceId, api) => memo(instanceId, (...args) => firstObservable(fn(api, ...args)));
}