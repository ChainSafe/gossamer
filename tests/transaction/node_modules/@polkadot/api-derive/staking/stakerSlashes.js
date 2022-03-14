// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
import { erasHistoricApplyAccount } from "./util.js";
export function _stakerSlashes(instanceId, api) {
  return memo(instanceId, (accountId, eras, withActive) => {
    const stakerId = api.registry.createType('AccountId', accountId).toString();
    return api.derive.staking._erasSlashes(eras, withActive).pipe(map(slashes => slashes.map(({
      era,
      nominators,
      validators
    }) => ({
      era,
      total: nominators[stakerId] || validators[stakerId] || api.registry.createType('Balance')
    }))));
  });
}
export const stakerSlashes = erasHistoricApplyAccount('_stakerSlashes');