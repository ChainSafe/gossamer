// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, of, switchMap } from 'rxjs';
import { memo } from "../util/index.js";
export function _members(instanceId, api) {
  return memo(instanceId, accountIds => combineLatest([of(accountIds), api.query.society.payouts.multi(accountIds), api.query.society.strikes.multi(accountIds), api.query.society.defenderVotes.multi(accountIds), api.query.society.suspendedMembers.multi(accountIds), api.query.society.vouching.multi(accountIds)]).pipe(map(([accountIds, payouts, strikes, defenderVotes, suspended, vouching]) => accountIds.map((accountId, index) => ({
    accountId,
    isDefenderVoter: defenderVotes[index].isSome,
    isSuspended: suspended[index].isTrue,
    payouts: payouts[index],
    strikes: strikes[index],
    vote: defenderVotes[index].unwrapOr(undefined),
    vouching: vouching[index].unwrapOr(undefined)
  })))));
}
/**
 * @description Get the member info for a society
 */

export function members(instanceId, api) {
  return memo(instanceId, () => api.query.society.members().pipe(switchMap(members => api.derive.society._members(members))));
}