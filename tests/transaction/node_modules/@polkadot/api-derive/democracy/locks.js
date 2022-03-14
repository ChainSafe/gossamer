// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of, switchMap } from 'rxjs';
import { BN_ZERO, isUndefined } from '@polkadot/util';
import { memo } from "../util/index.js";
const LOCKUPS = [0, 1, 2, 4, 8, 16, 32];

function parseEnd(api, vote, {
  approved,
  end
}) {
  return [end, approved.isTrue && vote.isAye || approved.isFalse && vote.isNay ? end.add((api.consts.democracy.voteLockingPeriod || api.consts.democracy.enactmentPeriod).muln(LOCKUPS[vote.conviction.index])) : BN_ZERO];
}

function parseLock(api, [referendumId, accountVote], referendum) {
  const {
    balance,
    vote
  } = accountVote.asStandard;
  const [referendumEnd, unlockAt] = referendum.isFinished ? parseEnd(api, vote, referendum.asFinished) : [BN_ZERO, BN_ZERO];
  return {
    balance,
    isDelegated: false,
    isFinished: referendum.isFinished,
    referendumEnd,
    referendumId,
    unlockAt,
    vote
  };
}

function delegateLocks(api, {
  balance,
  conviction,
  target
}) {
  return api.derive.democracy.locks(target).pipe(map(available => available.map(({
    isFinished,
    referendumEnd,
    referendumId,
    unlockAt,
    vote
  }) => ({
    balance,
    isDelegated: true,
    isFinished,
    referendumEnd,
    referendumId,
    unlockAt: unlockAt.isZero() ? unlockAt : referendumEnd.add((api.consts.democracy.voteLockingPeriod || api.consts.democracy.enactmentPeriod).muln(LOCKUPS[conviction.index])),
    vote: api.registry.createType('Vote', {
      aye: vote.isAye,
      conviction
    })
  }))));
}

function directLocks(api, {
  votes
}) {
  if (!votes.length) {
    return of([]);
  }

  return api.query.democracy.referendumInfoOf.multi(votes.map(([referendumId]) => referendumId)).pipe(map(referendums => votes.map((vote, index) => [vote, referendums[index].unwrapOr(null)]).filter(item => !!item[1] && isUndefined(item[1].end) && item[0][1].isStandard).map(([directVote, referendum]) => parseLock(api, directVote, referendum))));
}

export function locks(instanceId, api) {
  return memo(instanceId, accountId => api.query.democracy.votingOf ? api.query.democracy.votingOf(accountId).pipe(switchMap(voting => voting.isDirect ? directLocks(api, voting.asDirect) : voting.isDelegating ? delegateLocks(api, voting.asDelegating) : of([]))) : of([]));
}