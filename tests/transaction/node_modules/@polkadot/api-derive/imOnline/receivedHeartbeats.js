// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { combineLatest, map, of, switchMap } from 'rxjs';
import { BN_ZERO } from '@polkadot/util';
import { memo } from "../util/index.js";

function mapResult([result, validators, heartbeats, numBlocks]) {
  validators.forEach((validator, index) => {
    const validatorId = validator.toString();
    const blockCount = numBlocks[index];
    const hasMessage = !heartbeats[index].isEmpty;
    const prev = result[validatorId];

    if (!prev || prev.hasMessage !== hasMessage || !prev.blockCount.eq(blockCount)) {
      result[validatorId] = {
        blockCount,
        hasMessage,
        isOnline: hasMessage || blockCount.gt(BN_ZERO)
      };
    }
  });
  return result;
}
/**
 * @description Return a boolean array indicating whether the passed accounts had received heartbeats in the current session
 */


export function receivedHeartbeats(instanceId, api) {
  return memo(instanceId, () => {
    var _api$query$imOnline;

    return (_api$query$imOnline = api.query.imOnline) !== null && _api$query$imOnline !== void 0 && _api$query$imOnline.receivedHeartbeats ? api.derive.staking.overview().pipe(switchMap(({
      currentIndex,
      validators
    }) => combineLatest([of({}), of(validators), api.query.imOnline.receivedHeartbeats.multi(validators.map((_address, index) => [currentIndex, index])), api.query.imOnline.authoredBlocks.multi(validators.map(address => [currentIndex, address]))])), map(mapResult)) : of({});
  });
}