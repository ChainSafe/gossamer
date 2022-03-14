// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, switchMap } from 'rxjs';
import { firstMemo, memo } from "../util/index.js";
export function _stakerExposures(instanceId, api) {
  return memo(instanceId, (accountIds, eras, withActive = false) => {
    const stakerIds = accountIds.map(a => api.registry.createType('AccountId', a).toString());
    return api.derive.staking._erasExposure(eras, withActive).pipe(map(exposures => stakerIds.map(stakerId => exposures.map(({
      era,
      nominators: allNominators,
      validators: allValidators
    }) => {
      const isValidator = !!allValidators[stakerId];
      const validators = {};
      const nominating = allNominators[stakerId] || [];

      if (isValidator) {
        validators[stakerId] = allValidators[stakerId];
      } else if (nominating) {
        nominating.forEach(({
          validatorId
        }) => {
          validators[validatorId] = allValidators[validatorId];
        });
      }

      return {
        era,
        isEmpty: !Object.keys(validators).length,
        isValidator,
        nominating,
        validators
      };
    }))));
  });
}
export function stakerExposures(instanceId, api) {
  return memo(instanceId, (accountIds, withActive = false) => api.derive.staking.erasHistoric(withActive).pipe(switchMap(eras => api.derive.staking._stakerExposures(accountIds, eras, withActive))));
}
export const stakerExposure = firstMemo((api, accountId, withActive) => api.derive.staking.stakerExposures([accountId], withActive));