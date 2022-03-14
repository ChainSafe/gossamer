// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
export function votesOf(instanceId, api) {
  return memo(instanceId, accountId => api.derive.council.votes().pipe(map(votes => (votes.find(([from]) => from.eq(accountId)) || [null, {
    stake: api.registry.createType('Balance'),
    votes: []
  }])[1])));
}