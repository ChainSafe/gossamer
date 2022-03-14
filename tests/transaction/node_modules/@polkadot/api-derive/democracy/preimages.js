// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { isFunction } from '@polkadot/util';
import { firstMemo, memo } from "../util/index.js";

function isDemocracyPreimage(api, imageOpt) {
  return !!imageOpt && !api.query.democracy.dispatchQueue;
}

function constructProposal(api, [bytes, proposer, balance, at]) {
  let proposal;

  try {
    proposal = api.registry.createType('Proposal', bytes.toU8a(true));
  } catch (error) {
    console.error(error);
  }

  return {
    at,
    balance,
    proposal,
    proposer
  };
}

function parseDemocracy(api, imageOpt) {
  if (imageOpt.isNone) {
    return;
  }

  if (isDemocracyPreimage(api, imageOpt)) {
    const status = imageOpt.unwrap();

    if (status.isMissing) {
      return;
    }

    const {
      data,
      deposit,
      provider,
      since
    } = status.asAvailable;
    return constructProposal(api, [data, provider, deposit, since]);
  }

  return constructProposal(api, imageOpt.unwrap());
}

function getDemocracyImages(api, hashes) {
  return api.query.democracy.preimages.multi(hashes).pipe(map(images => images.map(imageOpt => parseDemocracy(api, imageOpt))));
}

export function preimages(instanceId, api) {
  return memo(instanceId, hashes => hashes.length ? isFunction(api.query.democracy.preimages) ? getDemocracyImages(api, hashes) : of([]) : of([]));
}
export const preimage = firstMemo((api, hash) => api.derive.democracy.preimages([hash]));