// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, of } from 'rxjs';
import { memo } from "../util/index.js"; // Voter is current tuple is 2.x-era

function isVoter(value) {
  return !Array.isArray(value);
}

function retrieveStakeOf(elections) {
  return elections.stakeOf.entries().pipe(map(entries => entries.map(([{
    args: [accountId]
  }, stake]) => [accountId, stake])));
}

function retrieveVoteOf(elections) {
  return elections.votesOf.entries().pipe(map(entries => entries.map(([{
    args: [accountId]
  }, votes]) => [accountId, votes])));
}

function retrievePrev(api, elections) {
  return combineLatest([retrieveStakeOf(elections), retrieveVoteOf(elections)]).pipe(map(([stakes, votes]) => {
    const result = [];
    votes.forEach(([voter, votes]) => {
      result.push([voter, {
        stake: api.registry.createType('Balance'),
        votes
      }]);
    });
    stakes.forEach(([staker, stake]) => {
      const entry = result.find(([voter]) => voter.eq(staker));

      if (entry) {
        entry[1].stake = stake;
      } else {
        result.push([staker, {
          stake,
          votes: []
        }]);
      }
    });
    return result;
  }));
}

function retrieveCurrent(elections) {
  return elections.voting.entries().pipe(map(entries => entries.map(([{
    args: [accountId]
  }, value]) => [accountId, isVoter(value) ? {
    stake: value.stake,
    votes: value.votes
  } : {
    stake: value[0],
    votes: value[1]
  }])));
}

export function votes(instanceId, api) {
  const elections = api.query.phragmenElection || api.query.electionsPhragmen || api.query.elections;
  return memo(instanceId, () => elections ? elections.stakeOf ? retrievePrev(api, elections) : retrieveCurrent(elections) : of([]));
}