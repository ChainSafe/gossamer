// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { switchMap } from 'rxjs';
import { memo } from "../util/index.js";
/**
 * @description Retrieve the staking overview, including elected and points earned
 */

export function currentPoints(instanceId, api) {
  return memo(instanceId, () => api.derive.session.indexes().pipe(switchMap(({
    activeEra
  }) => api.query.staking.erasRewardPoints(activeEra))));
}