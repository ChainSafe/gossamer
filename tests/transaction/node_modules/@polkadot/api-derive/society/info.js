// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";

/**
 * @description Get the overall info for a society
 */
export function info(instanceId, api) {
  return memo(instanceId, () => api.queryMulti([api.query.society.bids, api.query.society.defender, api.query.society.founder, api.query.society.head, api.query.society.maxMembers, api.query.society.pot]).pipe(map(([bids, defender, founder, head, maxMembers, pot]) => ({
    bids,
    defender: defender.unwrapOr(undefined),
    founder: founder.unwrapOr(undefined),
    hasDefender: defender.isSome && head.isSome && !head.eq(defender) || false,
    head: head.unwrapOr(undefined),
    maxMembers,
    pot
  }))));
}