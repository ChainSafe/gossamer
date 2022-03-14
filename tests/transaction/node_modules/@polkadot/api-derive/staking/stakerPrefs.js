// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
import { erasHistoricApplyAccount } from "./util.js";
export function _stakerPrefs(instanceId, api) {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  return memo(instanceId, (accountId, eras, _withActive) => api.query.staking.erasValidatorPrefs.multi(eras.map(e => [e, accountId])).pipe(map(all => all.map((validatorPrefs, index) => ({
    era: eras[index],
    validatorPrefs
  })))));
}
export const stakerPrefs = erasHistoricApplyAccount('_stakerPrefs');