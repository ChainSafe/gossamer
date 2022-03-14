// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, switchMap } from 'rxjs';
import { arrayFlatten } from '@polkadot/util';
import { memo } from "../util/index.js";
const DEFAULT_FLAGS = {
  withController: true,
  withExposure: true,
  withPrefs: true
};

function combineAccounts(nextElected, validators) {
  return arrayFlatten([nextElected, validators.filter(v => !nextElected.find(n => n.eq(v)))]);
}

export function electedInfo(instanceId, api) {
  return memo(instanceId, (flags = DEFAULT_FLAGS) => api.derive.staking.validators().pipe(switchMap(({
    nextElected,
    validators
  }) => api.derive.staking.queryMulti(combineAccounts(nextElected, validators), flags).pipe(map(info => ({
    info,
    nextElected,
    validators
  }))))));
}