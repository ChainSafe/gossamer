// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
/**
 * @description Get the member info for a society
 */

export function member(instanceId, api) {
  return memo(instanceId, accountId => api.derive.society._members([accountId]).pipe(map(([result]) => result)));
}