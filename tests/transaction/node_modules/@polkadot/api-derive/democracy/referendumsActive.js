// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { of, switchMap } from 'rxjs';
import { memo } from "../util/index.js";
export function referendumsActive(instanceId, api) {
  return memo(instanceId, () => api.derive.democracy.referendumIds().pipe(switchMap(ids => ids.length ? api.derive.democracy.referendumsInfo(ids) : of([]))));
}