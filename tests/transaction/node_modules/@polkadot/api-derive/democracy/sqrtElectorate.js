// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { bnSqrt } from '@polkadot/util';
import { memo } from "../util/index.js";
export function sqrtElectorate(instanceId, api) {
  return memo(instanceId, () => api.query.balances.totalIssuance().pipe(map(bnSqrt)));
}