// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
import { erasHistoricApplyAccount } from "./util.js";
export function _stakerPoints(instanceId, api) {
  return memo(instanceId, (accountId, eras, withActive) => {
    const stakerId = api.registry.createType('AccountId', accountId).toString();
    return api.derive.staking._erasPoints(eras, withActive).pipe(map(points => points.map(({
      era,
      eraPoints,
      validators
    }) => ({
      era,
      eraPoints,
      points: validators[stakerId] || api.registry.createType('RewardPoint')
    }))));
  });
}
export const stakerPoints = erasHistoricApplyAccount('_stakerPoints');